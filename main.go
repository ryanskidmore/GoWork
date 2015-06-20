package gowork

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"errors"
	"github.com/oleiade/lane"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"time"
)

var WorkersRegistrations map[int]*Worker
var NumWorkers int

type WorkServer struct {
	PresharedSecret string
	MongoSession    *mgo.Session
	Tables          *DatabaseTables
	Queue           *lane.Queue
	WorkResultChan  chan *WorkResult
}

type LocalWorker struct {
	PresharedSecret             string
	Id                          string `json:"Id"`
	SessionAuthenticationKey    string
	VerificationString          string `json:"Verification"`
	EncryptedVerificationString string `json:"EncryptedVerification"`
}

type DatabaseTables struct {
	WorkQueue     *mgo.Collection
	CompletedWork *mgo.Collection
}

type Worker struct {
	Id                       int
	VerificationString       string
	Registered               bool
	SessionAuthenticationKey string
}

type Work struct {
	Id        bson.ObjectId "_id"
	IdHex     string
	WorkJSON  string
	Timestamp time.Time
}

type WorkResult struct {
	Id         bson.ObjectId "_id"
	IdHex      string
	WorkObject *Work
	ResultJSON string
	Error      string
}

func NewServer(Secret string, Session *mgo.Session, Tables *DatabaseTables) *WorkServer {
	WorkQueue := lane.NewQueue()
	WorkersRegistrations = make(map[int]*Worker)
	NumWorkers = 0
	ResultChan := make(chan *WorkResult)
	WorkServerInst := &WorkServer{Secret, Session, Tables, WorkQueue, ResultChan}
	go AddResultToDB(WorkServerInst.WorkResultChan, WorkServerInst)
	return WorkServerInst
}

func NewWorker(Secret string, JSONToken string) (*LocalWorker, error) {
	LW := &LocalWorker{}
	LW.PresharedSecret = Secret
	err := json.Unmarshal([]byte(JSONToken), &LW)
	if err != nil {
		return &LocalWorker{}, errors.New("Failed to unmarshal token into register token:" + err.Error())
	}
	EncryptedVerification, err := encrypt([]byte(LW.PresharedSecret), []byte(LW.VerificationString))
	if err != nil {
		return &LocalWorker{}, errors.New("Failed to encrypt verification string:" + err.Error())
	}
	LW.EncryptedVerificationString = string(EncryptedVerification)
	return LW, nil
}

func CreateWork(WorkJSON string) *Work {
	NewWork := &Work{}
	ObjID := bson.NewObjectId()
	NewWork.IdHex = ObjID.Hex()
	NewWork.Timestamp = time.Now()
	NewWork.WorkJSON = WorkJSON
	return NewWork
}

func (ws WorkServer) AddWork(w *Work) error {
	LocalWork := w
	LocalWork.Timestamp = time.Now()
	LocalWorkMongo := LocalWork
	LocalWorkMongo.Id = bson.ObjectIdHex(LocalWorkMongo.IdHex)
	err := ws.Tables.WorkQueue.Insert(LocalWorkMongo)
	if err != nil {
		return errors.New("Could not insert work into MongoDB:" + err.Error())
	} else {
		ws.Queue.Enqueue(LocalWork)
	}
	return nil
}

func (ws WorkServer) GetWork(Id string, AuthenticationKey string) (*Work, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		return &Work{}, errors.New("Failed to convert Worker ID string to int:" + err.Error())
	}
	if WorkersRegistrations[IdInt].SessionAuthenticationKey == AuthenticationKey {
		WorkObj := ws.Queue.Dequeue()
		if WorkObj != nil {
			return WorkObj.(*Work), nil
		}
		return &Work{}, nil
	}
	return &Work{}, errors.New("Failed authentication")
}

func (ws WorkServer) SubmitCompleteWork(IdHex string, ResultJSON string, Error string) {
	WorkResultInst := &WorkResult{}
	WorkResultInst.Id = bson.ObjectIdHex(IdHex)
	WorkResultInst.IdHex = IdHex
	WorkResultInst.ResultJSON = ResultJSON
	WorkResultInst.Error = Error
	ws.WorkResultChan <- WorkResultInst
}

func AddResultToDB(wrc chan *WorkResult, ws *WorkServer) {
	for {
		WorkResultInst := <-wrc
		_ = ws.Tables.WorkQueue.FindId(WorkResultInst.Id).One(&WorkResultInst.WorkObject)
		_ = ws.Tables.WorkQueue.RemoveId(WorkResultInst.Id)
		_ = ws.Tables.CompletedWork.Insert(WorkResultInst)
	}
}

func (ws WorkServer) GetQueueSize() int {
	return ws.Queue.Size()
}

func (ws WorkServer) WorkerRegister() string {
	NewWorker := &Worker{}
	NumWorkers = NumWorkers + 1
	NewWorker.Id = NumWorkers
	NewWorker.VerificationString = uuid.New()
	NewWorker.Registered = false
	WorkersRegistrations[NewWorker.Id] = NewWorker
	return "{\"Id\":\"" + strconv.Itoa(NewWorker.Id) + "\", \"Verification\":\"" + NewWorker.VerificationString + "\"}"
}

func (ws WorkServer) WorkerVerify(Id string, EncryptedVerification string) (string, error) {
	IdInt, err := strconv.Atoi(Id)
	if err != nil {
		return "", errors.New("Failed to convert Worker ID string to int:" + err.Error())
	} else {
		DecryptedVerification, err := decrypt([]byte(ws.PresharedSecret), []byte(EncryptedVerification))
		if err != nil {
			return "", errors.New("Failed to decrypt worker verification string:" + err.Error())
		}
		if WorkersRegistrations[IdInt].VerificationString == string(DecryptedVerification) {
			WorkersRegistrations[IdInt].Registered = true
			WorkersRegistrations[IdInt].SessionAuthenticationKey = uuid.New()
			return WorkersRegistrations[IdInt].SessionAuthenticationKey, nil
		} else {
			return "", errors.New("Client key incorrect")
		}
	}
	return "", nil
}

func (lw LocalWorker) GetWork(WorkJSON string) (*Work, map[string]interface{}, error) { // *Work, map[string]interface{}, error
	WorkObj := &Work{}
	WorkParams := make(map[string]interface{})
	err := json.Unmarshal([]byte(WorkJSON), &WorkObj)
	if err != nil {
		return &Work{}, WorkParams, errors.New("Failed to unmarshal Work JSON:" + err.Error())
	}
	WorkObj.Id = bson.ObjectIdHex(WorkObj.IdHex)
	err = json.Unmarshal([]byte(WorkObj.WorkJSON), &WorkParams)
	if err != nil {
		return &Work{}, WorkParams, errors.New("Failed to unmarshal Work Params JSON:" + err.Error())
	}
	return WorkObj, WorkParams, nil
}

func (lw LocalWorker) UpdateWork(OriginalWork *Work, ResultJSON string, Error string) *WorkResult {
	Result := &WorkResult{}
	Result.Id = OriginalWork.Id
	Result.IdHex = OriginalWork.IdHex
	Result.WorkObject = OriginalWork
	Result.ResultJSON = ResultJSON
	Result.Error = Error
	return Result
}
