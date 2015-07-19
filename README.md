# GoWork

[![GoDoc](https://godoc.org/github.com/ryanskidmore/GoWork?status.svg)](https://godoc.org/github.com/ryanskidmore/GoWork)

GoWork is a library for the encapsulation and delivery of Work to a distributed set of Workers.

## Installation

`go get github.com/ryanskidmore/gowork`

#### NOTE: Only tested with Go 1.3+

## Features

* Worker Registration/Authentication
* Work Encapsulation and Tracking
* Built-in Work Queue
* Event based handler functions (e.g. run function when work complete, etc)

## Usage

### Server

#### Work

First step is to create a new work server,
```go
ws, err := gowork.NewServer("32 character secret")
```
 Error is returned when the secret isn't exactly 32 characters long.

Next, you can add parameters to be passed to your handler functions, though this is entirely optional.
```go
ws = ws.AddParams(map[string]interface{}{"param1":"value1"})
```

Then, you can add handler functions to be called when certain events happen.
```go
err = ws.NewHandler("event_id", func(e *Event, params map[string]interface{}){// function //})
```
This function will only return an error when the handler already exists, so in most situations you can probably ignore the err returned.

* Event ID's listed at the end of this section
* Event contains data about the event, and is of type gowork.Event. Contains data about Work, Worker, Error and timings.
* Params passed in earlier will be available in the params map (however, you will need to cast them into the appropriate type).

Now work can be created and added.
```go
w, err := gowork.CreateWork(WorkData interface{}, Timeout int64)
ws.Add(w)
```
WorkData can be anything that you want to pass to the worker. Timeout is duration in seconds in integer64 type until you want the work to Timeout.
Error is returned by `gowork.CreateWork()` when the WorkData fails to marshal into JSON format.

At this point, the work is created and added to the work queue.

The work is able to be retrieved then by calling the function `Get`
```go
w, err := ws.Get("Worker ID", "Worker Authentication String")
```
The Worker ID and Authentication String are provided to the worker by functions below in the Workers section. 

This function will return an error when:-

* The Worker ID cannot be converted to an integer
* The Work has timed out
* The Worker has failed to authenticate

When there is no work, the function will return an empty Work object and no error.

Then the work can be marshalled to be transferred to the worker (via a means of your choice) using the function `Marshal()` on the Work Object

```go
workstring := w.Marshal()
```

To accompany this function, there is also an unmarshal function to convert the string to a work object

```go
w := Unmarshal(workstring)
```

Once the worker has completed the work and passed it back to the server, it can be submitted via the `Submit()` function which marks the work as complete and calls the `work_complete` event.

```go
ws.Submit(w)
```

Should you wish to get the size of the work queue, there is a function available to return this.

```go
size := ws.QueueSize()
```

##### Event IDs
* add\_handler_error
* add_work
* get_work
* get\_work_empty
* get\_work_error
* work_complete
* work_timeout
* worker_registered
* worker_verify
* worker\_verify_error

#### Workers

When first receiving contact from the worker, the worker needs to be regisered. This provides the worker with an ID as well as a randomly generated string to encrypt in order to verify the secrets are the same on the client and the server, as well as to authenticate the worker.

```go
id, clienttest := ws.Workers.Register(ws)
```

The client then calls the appropriate functions on these strings and returns the encrypted string (client response) for verification to the server.

```go
authkey, err := ws.Workers.Verify(ws, id, clientresponse)
```
This function provides a session authentication key for the worker to use. Returns an error when:-

* Couldn't Worker ID to Int
* Failed to decrypt the client response
* Client Key is incorrect

### Client

When creating a new worker, the worker must first contact the work server and get an ID and Client Test before calling any Go Work functions. Once these two strings have been obtained, the following function can be called.

```go
Worker, err := gowork.NewWorker(Secret, Id, ClientTest)
```

This function will return an error when:-

* Secret is not 32 characters
* Couldn't convert Worker ID to Int
* Failed to encrypt the client test

This function will then store the client response in `Worker.Verification.ClientResponse`, which must then be passed to the server for verification, which will then return a session authentication key.

This key can be set within the worker by calling:-

```go
Worker = Worker.SetAuthenticationKey(Key)
```

Then the worker is ready to get work, this must be done independently of the library, using the Worker.Id and Worker.SessionAuthenticationKey strings for authentication. Once the work is retrieved, it can be unmarshalled into a work object and then passed into `Process`

```go
w, workdata, err := Worker.Process(w)
```

This function returns the original work object as well as the work data in a map of interfaces. An error will be returned when:-

* The work data cannot be unmarshalled
* The work has timed out

Once the worker has completed the work, it can be submitted using `Submit`

```go
w, err := Worker.Submit(w, result, error)
```

This function takes the original work object, appends the result to it (string) as well as any error (string) and returns the new work object back ready for marshalling and submission to the server. An error will be returned when the work has timed out.

## Handler Functions
A quick note on handlers, they're intended to act as middleware - so an ideal usage scenario would be adding a piece of work to a database when it's complete, for example.

## Contributors

Ryan Skidmore - [github.com/ryanskidmore](http://github.com/ryanskidmore)
