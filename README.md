
# SQS-Retry

Call third-party services by using AWS-SQS. How? Create a request, turn it into a JSON-string. That string send it to the queue and make sure that a listener will grab that request. Once the request arrives to the listener then, send that request to a concrete handler.

The handler should process the request by calling the third-party services. If the handler returns an error, then the listener will send back to the queue the request by increasing a delay-retry.

Check the file `example/example.go`
