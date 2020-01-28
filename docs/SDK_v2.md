# CloudEvents Golang SDK v2

## Terms

 - Binding
 - Bindings
 - Event
 - Message
 - Protocol
 - Transport
 - Client
 - Encoder
 - Decoder 
 - Transcoder
 - Codec
 

Poll Based Send - Does not make sense. 

## Push Based Send

```go
err := sender.Send(xx)
```
 
## Push Based Send with blocking Response

```go
event, err := sender.Send(xx) 
```

or 

```go
err := sender.Send(xx, responseFn) 
```
 
## Pull Based Receive

```go
event := receiver.Receive(xx)
```

## Push Based Receive

```go
func receiverFn(event)
```

## Push Based Receive with Synconus Response

```go
func receiverFn(event, responer)
```

or 

```go
func receiverFn(event) response
```


---

Expected Usage:

As a client:


```go
// Sync send
client.Send()

// Sync recieve
client.Recieve()

// Async Recive
client.Register(receiveFn)
```

As a protocol binder:

```go
// Sending
cehttp.ToRequest(event) request

// Recieving
cehttp.FromRequest(request) event
```

As middle ware:

```go
message = receiver.Recieve()
sender.Send(message)
```

Raw JSON:

```go
// encoding
bytes, err = json.Marshal(event)

// decoding
err = json.Unmarshal(bytes, event)
```

---

Original SDK

Client -> Transport -> Codec -> TransportImpl

       ^            ^         ^ 
    Events       Events    Message



(cloudevents.Event) --> Codec.Encode --> (transport.Message)
(transport.Message) --> Codec.Decode --> (cloudevents.Event)



New SDK

Message.Binary(encoder)
Message.Structured(encoder)
Message.Event(encoder)

Receiver.Recieve() message
Sender.Send(message)
Requester.Request(message) Receiver

!!--Confuses Builders and Encoders and Transcoders.

---

Assume error and context are in the correct locations.


What if there was two main ways to use the SDK:

1. As a user/client. Focused on producing and consuming events.

event = client.Recieve()
client.Send(target, event)
event = router.Filter(filter, event)


2. As a router. Focused on introspecting the attributes and forwarding.

message = router.Recieve()
router.Send(target, message)
message = router.Filter(filter, message)



This connected,

Objects Interfacing:
Client -> Router   -> ProtocolBinding
Data Interface:
Events -> Messages -> Bytes




