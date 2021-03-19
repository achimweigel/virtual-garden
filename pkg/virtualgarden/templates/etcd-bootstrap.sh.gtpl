#!/bin/sh
VALIDATION_MARKER={{ .DataDir }}/validation_marker

trap_and_propagate() {
  PID=$1
  shift
  for sig in "$@" ; do
  trap "kill -$sig $PID" "$sig"
  done
}

start_managed_etcd(){
  rm -rf $VALIDATION_MARKER
  etcd --config-file /bootstrap/etcd.conf.yml &
  ETCDPID=$!
  trap_and_propagate $ETCDPID INT TERM
  wait $ETCDPID
  RET=$?
  echo $RET > $VALIDATION_MARKER
  exit $RET
}

check_and_start_etcd(){
  while true;
  do
    wget http://localhost:{{ .BackupRestoreSidecarServicePort }}/initialization/status -S -O status;
    STATUS=`cat status`;
    case $STATUS in
    "New")
      wget http://localhost:{{ .BackupRestoreSidecarServicePort }}/initialization/start?mode=$1 -S -O - ;;
    "Progress")
      sleep 1;
      continue;;
    "Failed")
      sleep 1;
      continue;;
    "Successful")
      echo "Bootstrap preprocessing end time: $(date)"
      start_managed_etcd
      break
      ;;
    *)
      sleep 1;
      ;;
    esac;
  done
}

echo "Bootstrap preprocessing start time: $(date)"
# Do validation and bootstrap
if [ ! -f $VALIDATION_MARKER ] ;
then
  echo "No $VALIDATION_MARKER file. Perform complete initialization routine and start etcd."
  check_and_start_etcd full
else
  echo "$VALIDATION_MARKER file present. Check return status and decide on initialization"
  run_status=`cat $VALIDATION_MARKER`
  echo "$VALIDATION_MARKER content: $run_status"
  if [ $run_status == '143' ] || [ $run_status == '130' ] || [ $run_status == '0' ] ; then
    echo "Requesting sidecar to perform sanity validation"
    check_and_start_etcd sanity
  else
    echo "Requesting sidecar to perform full validation"
    check_and_start_etcd full
  fi
fi
