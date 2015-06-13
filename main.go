package gowork

import (
	"code.google.com/p/go-uuid/uuid"
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

func CreateWork(WorkJSON string) *Work {
	ObjID := bson.NewObjectId()
	return &Work{ObjID, ObjID.Hex(), WorkJSON, time.Now()}
}

func (ws WorkServer) AddWork(w *Work) error {
	LocalWork := w
	LocalWork.Timestamp = time.Now()
	err := ws.Tables.WorkQueue.Insert(LocalWork)
	if err != nil {
		return errors.New("Could not insert work into MongoDB:" + err.Error())
	} else {
		ws.Queue.Enqueue(w)
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
		} else {
			if WorkersRegistrations[IdInt].VerificationString == string(DecryptedVerification) {
				WorkersRegistrations[IdInt].Registered = true
				WorkersRegistrations[IdInt].SessionAuthenticationKey = uuid.New()
				return WorkersRegistrations[IdInt].SessionAuthenticationKey, nil
			}
		}
	}
	return "", nil
}

// CLIENT FUNCTIONS
