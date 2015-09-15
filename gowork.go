package gowork

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/oleiade/lane"
	"github.com/satori/go.uuid"
	"go.pedge.io/encrypt"
	"gopkg.in/mgo.v2/bson"
)

type WorkServer struct {
	Queue         *lane.Queue
	Handlers      map[string]func(*Event, map[string]interface{})
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
	EncodedSecret, err := encrypt.GenerateAESKey(encrypt.AES256Bits)
	if err != nil {
		return "", err
	}
	Secret, err := encrypt.DecodeString(EncodedSecret)
	if err != nil {
		return "", err
	}
	return string(Secret), nil
}

func NewEventError(msg string) *Event {
	return &Event{Error: msg, Time: time.Now().UTC().Unix()}
}

func NewEventWork(w *Work) *Event {
	return &Event{Work: w, Time: time.Now().UTC().Unix()}
}

func NewEventWorker(w *Worker) *Event {
	return &Event{Worker: w, Time: time.Now().UTC().Unix()}
}

func NewServer(Secret string) (*WorkServer, error) {
	Transformer, err := NewTransformer(Secret)
	if err != nil {
		return &WorkServer{}, err
	}
	return NewServerInit(Transformer), nil
}

func MustNewServer(Secret string) *WorkServer {
	return NewServerInit(MustNewTransformer(Secret))
}

func NewServerInit(Transformer encrypt.Transformer) *WorkServer {
	Queue := lane.NewQueue()
	WorkerMembers := make(map[int]*Worker)
	Workers := &WorkersStruct{WorkerMembers, Transformer, 0}
	HandlerFuncs := make(map[string]func(*Event, map[string]interface{}))
	HandlerParams := make(map[string]interface{})
	WorkServerInst := &WorkServer{Queue, HandlerFuncs, HandlerParams, Workers}
	return WorkServerInst
}

func (ws *WorkServer) NewHandler(event_id string, hf func(*Event, map[string]interface{})) error {
	if _, exists := ws.Handlers[event_id]; exists {
		ws.Event("add_handler_error", NewEventError("HandlerExists"))
		return errors.New("Handler already exists")
	}
	ws.Handlers[event_id] = hf
	return nil
}

func (ws *WorkServer) AddParams(params map[string]interface{}) *WorkServer {
	ws.HandlerParams = params
	return ws
}

func (ws *WorkServer) Event(event_id string, event *Event) {
	if handlerFunc, exists := ws.Handlers[event_id]; exists {
		handlerFunc(event, ws.HandlerParams)
	}
}

func (ws *WorkServer) Add(w *Work) {
	w.Time.Added = time.Now().UTC().Unix()
	ws.Event("add_work", NewEventWork(w))
	ws.Queue.Enqueue(w)
}

func (ws *WorkServer) Get(Id string, AuthenticationKey string) (*Work, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		ws.Event("get_work_error", NewEventError("StrconvError"))
		return &Work{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	}
	if ws.Workers.Members[IdInt].SessionAuthenticationKey != AuthenticationKey {
		ws.Event("get_work_error", NewEventError("AuthFailed"))
		return &Work{}, errors.New("Failed authentication")
	}
	WorkObj := ws.Queue.Dequeue()
	if WorkObj == nil {
		ws.Event("get_work_empty", NewEventError("NoWork"))
		return &Work{}, nil
	}
	if (WorkObj.(*Work).Time.Added + WorkObj.(*Work).Time.Timeout) > time.Now().UTC().Unix() {
		ws.Event("get_work", NewEventWork(WorkObj.(*Work)))
		return WorkObj.(*Work), nil
	}
	ws.Event("work_timeout", NewEventWork(WorkObj.(*Work)))
	return WorkObj.(*Work), errors.New("Work Timeout")
}

func (ws *WorkServer) Submit(w *Work) {
	if (w.Time.Added + w.Time.Timeout) <= time.Now().UTC().Unix() {
		w.Result.Error = "Timeout"
		w.Result.Status = "Timeout"
		ws.Event("work_timeout", NewEventWork(w))
		return
	}
	w.Id = bson.ObjectIdHex(w.IdHex)
	ws.Event("work_complete", NewEventWork(w))
}

func (ws *WorkServer) QueueSize() int {
	return ws.Queue.Size()
}

func (wrs *WorkersStruct) Register(ws *WorkServer) (string, string) {
	TempWC := wrs.WorkerCount
	wrs.WorkerCount += 1
	w := &Worker{
		Id:           TempWC + 1,
		Verification: &ClientTest{PlaintextVerification: uuid.NewV4().String()},
		Registered:   false,
	}
	wrs.Members[w.Id] = w
	ws.Event("worker_register", NewEventWorker(w))
	return strconv.Itoa(w.Id), w.Verification.PlaintextVerification
}

func (wrs *WorkersStruct) Verify(ws *WorkServer, Id string, Response string) (string, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		ws.Event("worker_verify_error", NewEventError("StrconvError"))
		return "", errors.New("Failed to convert Worker ID string to int:" + err.Error())
	}
	ClientResp, err := wrs.Transformer.Decrypt([]byte(Response))
	if err != nil {
		ws.Event("worker_verify_error", NewEventError("DecryptionError"))
		return "", errors.New("Failed to decrypt worker verification string:" + err.Error())
	}
	wrs.Members[IdInt].Verification.ClientResponse = string(ClientResp)
	if wrs.Members[IdInt].Verification.PlaintextVerification != string(wrs.Members[IdInt].Verification.ClientResponse) {
		ws.Event("worker_verify_error", NewEventError("KeyMismatch"))
		return "", errors.New("Client key incorrect")
	}
	wrs.Members[IdInt].Registered = true
	wrs.Members[IdInt].SessionAuthenticationKey = uuid.NewV4().String()
	ws.Event("worker_verify", NewEventWorker(wrs.Members[IdInt]))
	return wrs.Members[IdInt].SessionAuthenticationKey, nil
}

func NewWorker(Secret string, ID string, PlaintextVerification string) (*Worker, error) {
	wrk := &Worker{}
	Transformer, err := NewTransformer(Secret)
	if err != nil {
		return wrk, err
	}
	wrk.Transformer = Transformer
	wrk.Verification = &ClientTest{PlaintextVerification: PlaintextVerification}
	IdInt, err := strconv.Atoi(ID)
	if err != nil {
		return &Worker{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	}
	wrk.Id = IdInt
	ClientResponse, err := wrk.Transformer.Encrypt([]byte(wrk.Verification.PlaintextVerification))
	if err != nil {
		return &Worker{}, errors.New("Failed to encrypt verification string:" + err.Error())
	}
	wrk.Verification.ClientResponse = string(ClientResponse)
	return wrk, nil
}

func (wrk *Worker) SetAuthenticationKey(key string) *Worker {
	wrk.SessionAuthenticationKey = key
	return wrk
}

func (wrk *Worker) Process(w *Work) (*Work, map[string]interface{}, error) {
	WorkParams := make(map[string]interface{})
	if (w.Time.Added + w.Time.Timeout) <= time.Now().UTC().Unix() {
		return w, WorkParams, errors.New("Work Timeout")
	}
	err := json.Unmarshal([]byte(w.WorkJSON), &WorkParams)
	if err != nil {
		return w, WorkParams, errors.New("Failed to unmarshal Work Params JSON:" + err.Error())
	}
	w.Time.Recieved = time.Now().UTC().Unix()
	return w, WorkParams, nil
}

func (wrk *Worker) Submit(w *Work, ResultJSON string, Error string) (*Work, error) {
	wr := &WorkResult{}
	wr.ResultJSON = ResultJSON
	w.Time.Complete = time.Now().UTC().Unix()
	if (w.Time.Added + w.Time.Timeout) > time.Now().UTC().Unix() {
		wr.Error = Error
		wr.Status = "Complete"
		w.Result = wr
		return w, nil
	}
	wr.Error = "Timeout"
	wr.Status = "Timeout"
	w.Result = wr
	return w, errors.New("Timeout")
}

func CreateWork(WorkData interface{}, Timeout int64) (*Work, error) {
	NewWork := &Work{}
	NewWork.IdHex = bson.NewObjectId().Hex()
	NewWork.Result = &WorkResult{"", "Pending", ""}
	NewWork.Time = &TimeStats{Timeout: Timeout}
	WorkDataJSON, err := json.Marshal(WorkData)
	if err != nil {
		return &Work{}, errors.New("Failed to marshal work data:" + err.Error())
	}
	NewWork.WorkJSON = string(WorkDataJSON)
	return NewWork, nil
}

func (w *Work) Marshal() string {
	MarshalledWork, _ := json.Marshal(w)
	return string(MarshalledWork)
}

func Unmarshal(w string) *Work {
	WorkObject := &Work{}
	_ = json.Unmarshal([]byte(w), &WorkObject)
	return WorkObject
}

func NewTransformer(Secret string) (encrypt.Transformer, error) {
	if len(Secret) != 32 {
		return nil, fmt.Errorf("Length of secret must be 32, length was %d", len(Secret))
	}
	return encrypt.NewAESTransformer(encrypt.EncodeToString([]byte(Secret)))
}

func MustNewTransformer(Secret string) encrypt.Transformer {
	Transformer, err := NewTransformer(Secret)
	if err != nil {
		panic(err.Error())
	}
	return Transformer
}
