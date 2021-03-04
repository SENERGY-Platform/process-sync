package mongo

import (
	"context"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"reflect"
	"sync"
	"testing"
)

func TestDefinition(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoPort, _, err := docker.Mongo(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	config := configuration.Config{
		Debug:                            true,
		MongoUrl:                         "mongodb://localhost:" + mongoPort,
		MongoTable:                       "processes",
		MongoProcessDefinitionCollection: "process_definition",
		MongoDeploymentCollection:        "deployments",
		MongoProcessHistoryCollection:    "histories",
		MongoIncidentCollection:          "incidents",
		MongoProcessInstanceCollection:   "instances",
	}

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("create definition 1 n1", testCreateDefinition(db, "n1", "def1", "definition 1 n1"))
	t.Run("create definition 2 n1", testCreateDefinition(db, "n1", "def2", "definition 2 n1"))
	t.Run("create definition 1 n2", testCreateDefinition(db, "n2", "def1", "definition 1 n2"))
	t.Run("create definition 2 n2", testCreateDefinition(db, "n2", "def2", "definition 2 n2"))

	t.Run("read definition 1 n1", testReadDefinition(db, "n1", "def1", "definition 1 n1"))

	t.Run("list n1", testListDefinition(db, []string{"n1"}, []model.ProcessDefinition{
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def2",
				Name: "definition 2 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
	}))

	t.Run("list n1 n2", testListDefinition(db, []string{"n1", "n2"}, []model.ProcessDefinition{
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n2",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def2",
				Name: "definition 2 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def2",
				Name: "definition 2 n2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n2",
			},
		},
	}))

	t.Run("remove unknown n2 def1", testRemoveUnknownDefinitions(db, "n2", []string{"def1", "def3", "def4"}))

	t.Run("list n1 n2", testListDefinition(db, []string{"n1", "n2"}, []model.ProcessDefinition{
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n2",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def2",
				Name: "definition 2 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
	}))

	t.Run("remove n2 def1", testRemoveDefinition(db, "n2", "def1"))

	t.Run("list n1 n2", testListDefinition(db, []string{"n1", "n2"}, []model.ProcessDefinition{
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def1",
				Name: "definition 1 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
		{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   "def2",
				Name: "definition 2 n1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId: "n1",
			},
		},
	}))
}

func testRemoveDefinition(db *Mongo, networkId string, definitionId string) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.RemoveProcessDefinition(networkId, definitionId)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testRemoveUnknownDefinitions(db *Mongo, networkId string, known []string) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.RemoveUnknownProcessDefinitions(networkId, known)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testListDefinition(db *Mongo, networkIds []string, expected []model.ProcessDefinition) func(t *testing.T) {
	return func(t *testing.T) {
		actual, err := db.ListProcessDefinitions(networkIds, 10, 0, "name")
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Error(actual, expected)
			return
		}
	}
}

func testReadDefinition(db *Mongo, networkId string, definitionId string, name string) func(t *testing.T) {
	return func(t *testing.T) {
		definition, err := db.ReadProcessDefinition(networkId, definitionId)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(definition, model.ProcessDefinition{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   definitionId,
				Name: name,
			},
			SyncInfo: model.SyncInfo{
				NetworkId: networkId,
			},
		}) {
			t.Error(definition)
			return
		}
	}
}

func testCreateDefinition(db *Mongo, networkId string, defId string, name string) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.SaveProcessDefinition(model.ProcessDefinition{
			ProcessDefinition: camundamodel.ProcessDefinition{
				Id:   defId,
				Name: name,
			},
			SyncInfo: model.SyncInfo{
				NetworkId: networkId,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
	}
}
