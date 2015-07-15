package gowork

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/oleiade/lane"
	"github.com/peter-edge/go-encrypt"
	"gopkg.in/mgo.v2/bson"
)

type WorkServer struct {
	Queue         *lane.Queue
	Handlers      map[string]interface{}
	HandlerParams map[string]interface{}
	Workers       *WorkersStruct
}

type WorkersStruct struct {
	Members     map[int]*Worker
	Transformer encrypt.Transformer
	WorkerCount int
}

type Worker struct {
	Id                       int
	Registered               bool
	Transformer              encrypt.Transformer
	SessionAuthenticationKey string
	Verification             *ClientTest
}

type ClientTest struct {
	PlaintextVerification string `json:"Verification"`
	ClientResponse        string `json:"Response"`
}

type Work struct {
	Id       bson.ObjectId `json:"-" bson:"_id"`
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
	Timeout  int64
}

type Event struct {
	Work   *Work
	Worker *Worker
	Error  string
	Time   int64
}

func GenerateSecret() (string, error) {
	encodedSecret, err := encrypt.GenerateAESKey(encrypt.AES256Bits)
	if err != nil {
		return "", err
	}
	secret, err := encrypt.DecodeString(encodedSecret)
	if err != nil {
		return "", err
	}
	return string(secret), nil
}

func NewServer(Secret string) (*WorkServer, error) {
	transformer, err := newTransformer(Secret)
	if err != nil {
		return &WorkServer{}, err
	}
	Queue := lane.NewQueue()
	WorkerMembers := make(map[int]*Worker)
	Workers := &WorkersStruct{WorkerMembers, transformer, 0}
	HandlerFuncs := make(map[string]interface{})
	HandlerParams := make(map[string]interface{})
	WorkServerInst := &WorkServer{Queue, HandlerFuncs, HandlerParams, Workers}
	return WorkServerInst, nil
}

func (ws WorkServer) NewHandler(event_id string, hf func(*Event, map[string]interface{})) error {
	if _, exists := ws.Handlers[event_id]; exists {
		ws.Event("add_handler_error", &Event{Error: "HandlerExists", Time: time.Now().UTC().Unix()})
		return errors.New("Handler already exists")
	} else {
		ws.Handlers[event_id] = hf
		return nil
	}
}

func (ws WorkServer) AddParams(params map[string]interface{}) *WorkServer {
	ws.HandlerParams = params
	return &ws
}

func (ws WorkServer) Event(event_id string, event *Event) {
	if handlerFunc, exists := ws.Handlers[event_id]; exists {
		handlerFunc.(func(*Event, map[string]interface{}))(event, ws.HandlerParams)
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
				if (WorkObj.(*Work).Time.Added + WorkObj.(*Work).Time.Timeout) > time.Now().UTC().Unix() {
					ws.Event("get_work", &Event{Work: WorkObj.(*Work), Time: time.Now().UTC().Unix()})
					return WorkObj.(*Work), nil
				} else {
					ws.Event("work_timeout", &Event{Work: WorkObj.(*Work), Time: time.Now().UTC().Unix()})
					return WorkObj.(*Work), errors.New("Work Timeout")
				}

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

func (ws WorkServer) Submit(w *Work) {
	if (w.Time.Added + w.Time.Timeout) > time.Now().UTC().Unix() {
		w.Id = bson.ObjectIdHex(w.IdHex)
		ws.Event("work_complete", &Event{Work: w, Time: time.Now().UTC().Unix()})
	} else {
		w.Result.Error = "Timeout"
		w.Result.Status = "Timeout"
		ws.Event("work_timeout", &Event{Work: w, Time: time.Now().UTC().Unix()})
	}
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
		ClientResp, err := wrs.Transformer.Decrypt([]byte(Response))
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
	transformer, err := newTransformer(Secret)
	if err != nil {
		return wrk, err
	}
	wrk.Transformer = transformer
	wrk.Verification = &ClientTest{PlaintextVerification: PlaintextVerification}
	IdInt, err := strconv.Atoi(ID)
	if err != nil {
		return &Worker{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	} else {
		wrk.Id = IdInt
		ClientResponse, err := wrk.Transformer.Encrypt([]byte(wrk.Verification.PlaintextVerification))
		if err != nil {
			return &Worker{}, errors.New("Failed to encrypt verification string:" + err.Error())
		} else {
			wrk.Verification.ClientResponse = string(ClientResponse)
			return wrk, nil
		}
	}
}

func (wrk Worker) SetAuthenticationKey(key string) *Worker {
	wrk.SessionAuthenticationKey = key
	return &wrk
}

func (wrk Worker) Process(w *Work) (*Work, map[string]interface{}, error) {
	WorkParams := make(map[string]interface{})
	if (w.Time.Added + w.Time.Timeout) > time.Now().UTC().Unix() {
		err := json.Unmarshal([]byte(w.WorkJSON), &WorkParams)
		if err != nil {
			return w, WorkParams, errors.New("Failed to unmarshal Work Params JSON:" + err.Error())
		} else {
			w.Time.Recieved = time.Now().UTC().Unix()
			return w, WorkParams, nil
		}
	} else {
		return w, WorkParams, errors.New("Work Timeout")
	}
}

func (wrk Worker) Submit(w *Work, ResultJSON string, Error string) (*Work, error) {
	wr := &WorkResult{}
	wr.ResultJSON = ResultJSON
	w.Time.Complete = time.Now().UTC().Unix()
	if (w.Time.Added + w.Time.Timeout) > time.Now().UTC().Unix() {
		wr.Error = Error
		wr.Status = "Complete"
		w.Result = wr
		return w, nil
	} else {
		wr.Error = "Timeout"
		wr.Status = "Timeout"
		w.Result = wr
		return w, errors.New("Timeout")
	}
}

func CreateWork(WorkData interface{}, Timeout int64) (*Work, error) {
	NewWork := &Work{}
	NewWork.IdHex = bson.NewObjectId().Hex()
	NewWork.Result = &WorkResult{"", "Pending", ""}
	NewWork.Time = &TimeStats{Timeout: Timeout}
	WorkDataJSON, err := json.Marshal(WorkData)
	if err != nil {
		return &Work{}, errors.New("Failed to marshal work data:" + err.Error())
	} else {
		NewWork.WorkJSON = string(WorkDataJSON)
		return NewWork, nil
	}
}

func (w Work) Marshal() string {
	MarshalledWork, _ := json.Marshal(w)
	return string(MarshalledWork)
}

func Unmarshal(w string) *Work {
	WorkObject := &Work{}
	_ = json.Unmarshal([]byte(w), &WorkObject)
	return WorkObject
}

func newTransformer(secret string) (encrypt.Transformer, error) {
	if len(secret) != 32 {
		return nil, fmt.Errorf("length of secret must be 32, length was %d", len(secret))
	}
	return encrypt.NewAESTransformer(encrypt.EncodeToString([]byte(secret)))
}
