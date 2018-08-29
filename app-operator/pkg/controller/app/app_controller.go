package app

import (
	"context"
	"log"

	appv1alpha1 "github.com/operator-framework/operator-sdk-samples/app-operator/pkg/apis/app/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new App Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApp{client: mgr.GetClient(), cache: mgr.GetCache(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// nsPredicate := predicateForNamespace(namespace)

	// Create a new controller
	c, err := controller.New("app-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to App
	err = c.Watch(&source.Kind{Type: &appv1alpha1.App{}}, &handler.EnqueueRequestForObject{}) //, nsPredicate)
	if err != nil {
		return err
	}

	// Watch for Pods created by App
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.App{},
	}) //, nsPredicate)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApp{}

// ReconcileApp reconciles a App object
type ReconcileApp struct {
	cache  cache.Cache
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a App object and makes changes based on the state read
// and what is in the App.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding creates a
// busy-box pod as an example
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the App instance from the cache
	instance := &appv1alpha1.App{}
	err := r.cache.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define the desired Pod object
	pod := newbusyBoxPod(instance)
	// Set App instance as owner
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating Pod %s/%s\n", pod.Namespace, pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			log.Printf("Could not find Pod %v.\n", request)
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Printf("Could not fetch Pod %v.\n", request)
		return reconcile.Result{}, err
	}

	// Update the found object and write the result back if there are any changes
	// if !reflect.DeepEqual(pod.Spec, found.Spec) {
	// 	found.Spec = pod.Spec
	// 	log.Printf("Updating Pod %s/%s\n", pod.Namespace, pod.Name)
	// 	err = r.client.Update(context.TODO(), found)
	// 	if err != nil {
	// 		log.Printf("Could not update Pod %v: %v\n", request, err)
	// 		return reconcile.Result{}, err
	// 	}
	// }
	return reconcile.Result{}, nil
}

// newbusyBoxPod demonstrates how to create a busybox pod
func newbusyBoxPod(cr *appv1alpha1.App) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}

// TODO: Should be a util in either the SDK or the controller-runtime
// predicateForNamespace returns a predicate that only allows events from the desired namespace
func predicateForNamespace(namespace string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Meta.GetNamespace() == namespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaNew.GetNamespace() == namespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Meta.GetNamespace() == namespace
		},
	}
}
