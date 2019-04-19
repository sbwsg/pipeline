while true ; do
	clear
	kubectl get taskruns -o wide -n quotatest
	# kubectl get pods -o wide -n quotatest
	sleep 2
done
