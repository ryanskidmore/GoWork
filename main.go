package gowork

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"errors"
	"github.com/oleiade/lane"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"time"
)

// TODO

// Timeouts

type WorkServer struct {
	Queue    *lane.Queue
	Handlers map[string]interface{}
	Workers  *WorkersStruct
}

type WorkersStruct struct {
	Members         map[int]*Worker
	PresharedSecret string
	WorkerCount     int
}

type Worker struct {
	Id                       int `json:"Id"`
	Registered               bool
	PresharedSecret          string
	SessionAuthenticationKey string `json:"SessionAuthenticationKey"`
	Verification             *ClientTest
}

type ClientTest struct {
	PlaintextVerification string `json:"Verification"`
	ClientResponse        string `json:"Response"`
}

type Work struct {
	Id       bson.ObjectId "_id"
	IdHex    string
	WorkJSON string
	Result   *WorkResult
	Time     *TimeStats
}

type WorkResult struct {
	ResultJSON string
	Status     string
	Error      string
}

type TimeStats struct {
	Added    int64
	Recieved int64
	Complete int64
}

type Event struct {
	Work   *Work
	Worker *Worker
	Error  string
	Time   int64
}

func NewServer(Secret string) *WorkServer {
	Queue := lane.NewQueue()
	WorkerMembers := make(map[int]*Worker)
	Workers := &WorkersStruct{WorkerMembers, Secret, 0}
	HandlerFuncs := make(map[string]interface{})
	WorkServerInst := &WorkServer{Queue, HandlerFuncs, Workers}
	return WorkServerInst
}

func (ws WorkServer) NewHandler(event_id string, hf func(*Event)) error {
	if _, exists := ws.Handlers[event_id]; exists {
		ws.Event("add_handler_error", &Event{Error: "HandlerExists", Time: time.Now().UTC().Unix()})
		return errors.New("Handler already exists")
	} else {
		ws.Handlers[event_id] = hf
		return nil
	}
}

func (ws WorkServer) Event(event_id string, event *Event) {
	if handlerFunc, exists := ws.Handlers[event_id]; exists {
		handlerFunc.(func(*Event))(event)
	}
}

func (ws WorkServer) Add(w *Work) {
	w.Time.Added = time.Now().UTC().Unix()
	ws.Event("add_work", &Event{Work: w, Time: time.Now().UTC().Unix()})
	ws.Queue.Enqueue(w)
}

func (ws WorkServer) Get(Id string, AuthenticationKey string) (*Work, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		ws.Event("get_work_error", &Event{Error: "StrconvError", Time: time.Now().UTC().Unix()})
		return &Work{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	} else {
		if ws.Workers.Members[IdInt].SessionAuthenticationKey == AuthenticationKey {
			WorkObj := ws.Queue.Dequeue()
			if WorkObj != nil {
				ws.Event("get_work", &Event{Work: WorkObj.(*Work), Time: time.Now().UTC().Unix()})
				return WorkObj.(*Work), nil
			} else {
				ws.Event("get_work_empty", &Event{Error: "NoWork", Time: time.Now().UTC().Unix()})
				return &Work{}, nil
			}
		} else {
			ws.Event("get_work_error", &Event{Error: "AuthFailed", Time: time.Now().UTC().Unix()})
			return &Work{}, errors.New("Failed authentication")
		}
	}
}

func (ws WorkServer) Submit(w *Work, wres *WorkResult) {
	w.Result = wres
	w.Id = bson.ObjectIdHex(w.IdHex)
	w.Time.Complete = time.Now().UTC().Unix()
	ws.Event("work_complete", &Event{Work: w, Time: time.Now().UTC().Unix()})
}

func (ws WorkServer) QueueSize() int {
	return ws.Queue.Size()
}

func (wrs WorkersStruct) Register(ws *WorkServer) (string, string) {
	TempWC := wrs.WorkerCount
	wrs.WorkerCount += 1
	NewWorker := &Worker{}
	NewWorker.Id = TempWC + 1
	NewWorker.Verification = &ClientTest{PlaintextVerification: uuid.New()}
	NewWorker.Registered = false
	wrs.Members[NewWorker.Id] = NewWorker
	ws.Event("worker_register", &Event{Worker: NewWorker, Time: time.Now().UTC().Unix()})
	return strconv.Itoa(NewWorker.Id), NewWorker.Verification.PlaintextVerification
}

func (wrs WorkersStruct) Verify(ws *WorkServer, Id string, Response string) (string, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		ws.Event("worker_verify_error", &Event{Error: "StrconvError", Time: time.Now().UTC().Unix()})
		return "", errors.New("Failed to convert Worker ID string to int:" + err.Error())
	} else {
		ClientResp, err := decrypt([]byte(wrs.PresharedSecret), []byte(Response))
		if err != nil {
			ws.Event("worker_verify_error", &Event{Error: "DecryptionError", Time: time.Now().UTC().Unix()})
			return "", errors.New("Failed to decrypt worker verification string:" + err.Error())
		} else {
			wrs.Members[IdInt].Verification.ClientResponse = string(ClientResp)
			if wrs.Members[IdInt].Verification.PlaintextVerification == string(wrs.Members[IdInt].Verification.ClientResponse) {
				wrs.Members[IdInt].Registered = true
				wrs.Members[IdInt].SessionAuthenticationKey = uuid.New()
				ws.Event("worker_verify", &Event{Worker: wrs.Members[IdInt], Time: time.Now().UTC().Unix()})
				return wrs.Members[IdInt].SessionAuthenticationKey, nil
			} else {
				ws.Event("worker_verify_error", &Event{Error: "KeyMismatch", Time: time.Now().UTC().Unix()})
				return "", errors.New("Client key incorrect")
			}
		}
	}
	ws.Event("worker_verify_error", &Event{Error: "UnknownError", Time: time.Now().UTC().Unix()})
	return "", nil
}

func NewWorker(Secret string, ID string, PlaintextVerification string) (*Worker, error) {
	wrk := &Worker{}
	wrk.PresharedSecret = Secret
	wrk.Verification = &ClientTest{PlaintextVerification: PlaintextVerification}
	IdInt, err := strconv.Atoi(ID)
	if err != nil {
		return &Worker{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	} else {
		wrk.Id = IdInt
		ClientResponse, err := encrypt([]byte(wrk.PresharedSecret), []byte(wrk.Verification.PlaintextVerification))
		if err != nil {
			return &Worker{}, errors.New("Failed to encrypt verification string:" + err.Error())
		} else {
			wrk.Verification.ClientResponse = string(ClientResponse)
			return wrk, nil
		}
	}
}

func (wrk Worker) Get(w *Work) (*Work, map[string]interface{}, error) {
	w.Id = bson.ObjectIdHex(w.IdHex)
	WorkParams := make(map[string]interface{})
	err := json.Unmarshal([]byte(w.WorkJSON), &WorkParams)
	if err != nil {
		return &Work{}, WorkParams, errors.New("Failed to unmarshal Work Params JSON:" + err.Error())
	} else {
		w.Time.Recieved = time.Now().UTC().Unix()
		return w, WorkParams, nil
	}
}

func (wrk Worker) Submit(w *Work, ResultJSON string, Error string) *Work {
	wr := &WorkResult{}
	wr.ResultJSON = ResultJSON
	wr.Error = Error
	wr.Status = "Complete"
	w.Result = wr
	return w
}

func CreateWork(WorkData interface{}) (*Work, error) { // allow passing of interface then marshal it
	NewWork := &Work{}
	NewWork.IdHex = bson.NewObjectId().Hex()
	WorkDataJSON, err := json.Marshal(WorkData)
	if err != nil {
		return &Work{}, errors.New("Failed to marshal work data:" + err.Error())
	} else {
		NewWork.WorkJSON = string(WorkDataJSON)
		return NewWork, nil
	}
}
