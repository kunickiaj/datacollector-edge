# HTTP Server to Trash

### To start pipeline on SDE start

    $ bin/edge -start=httpServerToTensorToHttp

### To pass runtime parameters

    $ bin/edge -start=httpServerToTrash -runtimeParameters='{"serverAppId": "tensorFlowReceiver", "clientAppId":"tensorFlowSender","clientHttpUrl":"http://localhost:20000","serverPort":"10000","modelPath":"/home/santhosh_activity_tracker"}'

## REST API

    $ curl -X GET http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/status
    $ curl -X POST http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/start
    $ curl -X POST http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/stop
    $ curl -X POST http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/resetOffset
    $ curl -X GET http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/metrics

### To pass runtime parameters during start

    $ curl -X POST http://localhost:18633/rest/v1/pipeline/httpServerToTensorToHttp/start -H 'Content-Type: application/json;charset=UTF-8' --data-binary {"serverAppId": "tensorFlowReceiver", "clientAppId":"tensorFlowSender","clientHttpUrl":"http://localhost:20000","serverPort":"10000","modelPath":"/home/santhosh_activity_tracker"}'

## SDC Edge Pipeline

![Image of SDC Edge Pipeline](edge.png)

