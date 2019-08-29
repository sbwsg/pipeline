/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sidecars

import (
	"errors"
	"flag"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	nopImage = flag.String("nop-image", "override-with-nop:latest", "The container image used to kill sidecars")
)

type UpdatePod func(*corev1.Pod) (*corev1.Pod, error)

func Stop(clientset kubernetes.Interface, pod *corev1.Pod) error {
	if err := attemptGracefulShutdown(pod, clientset); err != nil {
		return forceStop(pod, clientset.CoreV1().Pods(pod.Namespace).Update)
	}
	return nil
}

func attemptGracefulShutdown(pod *corev1.Pod, clientset kubernetes.Interface) error {
	// TODO:
	//
	// 1. Exec kill in sidecar to send SIGTERM to PID 1
	// 2. After termination grace period (30s default) exec kill again to send
	//    SIGKILL to PID 1
	// 3. If exec'ing kill doesn't work in step 1 or the sidecar doesn't stop
	//    after step 2 do a "nop swap" of the sidecar's image field and
	//    forcefully shut it down.
	//
	// for _, container := range pod.Containers {
	// 	clientset.RESTClient().Post().Namespace(pod.Namespace).Resource("pods").Name(pod.Name).SubResource("exec").VersionParams(
	// 		corev1.PodExecOptions{
	// 			// 15 is SIGTERM
	// 			Command:   []string{"kill", "-15", "1"},
	// 			Stdin:     false,
	// 			Stdout:    true,
	// 			Stderr:    true,
	// 			TTY:       true,
	// 			Container: container.Name,
	// 		}, scheme.ParameterCodex)
	// }
	return errors.New("unimplemented")
}

// forceStop stops all sidecar containers inside a pod. A container is considered
// to be a sidecar if it is currently running. This func is only expected to
// be called after a TaskRun completes and all Step containers Step containers
// have already stopped.
//
// A sidecar is killed by replacing its current container image with the nop
// image, which in turn quickly exits. If the sidecar defines a command then
// it will exit with a non-zero status. When we check for TaskRun success we
// have to check for the containers we care about - not the final Pod status.
func forceStop(pod *corev1.Pod, updatePod UpdatePod) error {
	updated := true
	if pod.Status.Phase == corev1.PodRunning {
		for _, s := range pod.Status.ContainerStatuses {
			if s.State.Running != nil {
				for j, c := range pod.Spec.Containers {
					if c.Name == s.Name && c.Image != *nopImage {
						updated = true
						pod.Spec.Containers[j].Image = *nopImage
					}
				}
			}
		}
	}
	if updated {
		if _, err := updatePod(pod); err != nil {
			return err
		}
	}
	return nil
}
