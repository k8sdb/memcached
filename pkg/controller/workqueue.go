package controller

import (
	"fmt"
	"reflect"
	"time"

	"github.com/appscode/go/log"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Controller) initWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (rt.Object, error) {
			return c.ExtClient.Memcacheds(metav1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.ExtClient.Memcacheds(metav1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}

	// create the workqueue
	c.queue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "memcached")

	// stored deleted objects
	c.deletedIndexer = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})

	keyExists = make(map[string]bool)

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the Memcached key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Memcached than the version which was responsible for triggering the update.
	c.indexer, c.informer = cache.NewIndexerInformer(lw, &api.Memcached{}, c.syncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if exists, _ := keyExists[key]; err == nil && !exists {
				c.queue.Add(key)
				c.deletedIndexer.Delete(obj)
				keyExists[key] = true
			} else if exists {
				log.Debugf("Key:", key, "already processing. Not added new key")
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.deletedIndexer.Add(obj)
				c.queue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldObj, ok := old.(*api.Memcached)
			if !ok {
				log.Errorln("Invalid Memcached object")
				return
			}
			newObj, ok := new.(*api.Memcached)
			if !ok {
				log.Errorln("Invalid Memcached object")
				return
			}
			if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if exists, _ := keyExists[key]; err == nil && !exists {
					c.queue.Add(key)
					keyExists[key] = true
				} else if exists {
					log.Debugf("Key:", key, "already processing. Not added updated key")
				}
			}
		},
	}, cache.Indexers{})
}

func (c *Controller) runWatcher(threadiness int, stopCh chan struct{}) {

	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Infof("Starting Memcached controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Infof("Stopping Memcached controller")

}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two Memcacheds with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.runMemcachedInjector(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		log.Debugf("Setting keyExists to false.")
		keyExists[key.(string)] = false
		return true
	}
	log.Errorf("Failed to process Memcached %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < c.opt.MaxNumRequeues {
		log.Infof("Error syncing crd %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return true
	}

	c.queue.Forget(key)
	log.Debugf("Setting keyExists to false.")
	keyExists[key.(string)] = false
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

func (c *Controller) runMemcachedInjector(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		log.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		if obj, exists, err = c.deletedIndexer.GetByKey(key); err == nil && exists {
			memcached := obj.(*api.Memcached)
			if err := c.pause(memcached.DeepCopy()); err != nil {
				log.Errorln(err)
			}
			c.deletedIndexer.Delete(key)
		}
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Memcached was recreated with the same name
		memcached := obj.(*api.Memcached)
		if err := c.create(memcached.DeepCopy()); err != nil {
			log.Errorln(err)
			c.pushFailureEvent(memcached, err.Error())
		}
	}
	return nil
}
