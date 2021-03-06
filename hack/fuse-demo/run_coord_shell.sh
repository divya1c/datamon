#! /bin/zsh

container_name=demo-app

while getopts s opt; do
    case $opt in
        (s)
            container_name='datamon-sidecar'
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))


pod_name=$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')

if [[ -z $pod_name ]]; then
	echo 'coord demo pod not found' 1>&2
	exit 1
fi

kubectl exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/bash"
