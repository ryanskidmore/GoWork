package main

import (
	"fmt"
	"github.com/levigross/grequests"
	"github.com/ryanskidmore/GoWork"
	"strings"
)

func main() {
	response, err := grequests.Get("http://127.0.0.1:3000/register", nil)
	if err != nil {
		panic("Unable to register:" + err.Error())
	}
	respdata := strings.Split(response.String(), ",")
	id := respdata[0]
	clienttest := respdata[1]
	worker, err := gowork.NewWorker("w4PYxQjVP9ZStjWpBt5t28CEBmRs8NPx", id, clienttest)
	testresponse, err := grequests.Post("http://127.0.0.1:3000/verify", &grequests.RequestOptions{Params: map[string]string{"id": id, "clientresp": worker.Verification.ClientResponse}})
	if err != nil {
		panic("Unable to verify:" + err.Error())
	}
	worker = worker.SetAuthenticationKey(testresponse.String())
	fmt.Println(worker.SessionAuthenticationKey)
	testresponse, err = grequests.Post("http://127.0.0.1:3000/get_work", &grequests.RequestOptions{Params: map[string]string{"id": id, "sessionauthkey": worker.SessionAuthenticationKey}})
	fmt.Println(testresponse.String())
}
