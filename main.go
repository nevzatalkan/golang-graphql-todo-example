package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/mnmtanish/go-graphiql"
)

import (
	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
)

var engine *xorm.Engine

type Todo struct {
	Id      int64
	Text    string
	Done    bool
	Version int `xorm:"version"` // Optimistic Locking
}

var TodoList []Todo
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int64) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func init() {
	todo1 := Todo{Id: 1, Text: "A todo not to forget", Done: false}
	todo2 := Todo{Id: 2, Text: "This is the most important", Done: false}
	todo3 := Todo{Id: 3, Text: "Please do this or else", Done: false}
	TodoList = append(TodoList, todo1, todo2, todo3)

	rand.Seed(time.Now().UnixNano())
}

// define custom GraphQL ObjectType `todoType` for our Golang struct `Todo`
// Note that
// - the fields in our todoType maps with the json tags for the fields in our struct
// - the field type matches the field type in our struct
var todoType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Todo",
	Fields: graphql.Fields{
		"Id": &graphql.Field{
			Type: graphql.Int,
		},
		"Text": &graphql.Field{
			Type: graphql.String,
		},
		"Done": &graphql.Field{
			Type: graphql.Boolean,
		},
	},
})

// root mutation
var rootMutation = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootMutation",
	Fields: graphql.Fields{
		/*
			curl -g 'http://localhost:8081/graphql?query=mutation+_{createTodo(Text:"My+new+todo"){Id,Text,Done}}'
		*/
		"createTodo": &graphql.Field{
			Type:        todoType, // the return type for this field
			Description: "Create new todo",
			Args: graphql.FieldConfigArgument{
				"Text": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {

				// marshall and cast the argument value
				Text, _ := params.Args["Text"].(string)

				// figure out new Id
				var newID int64 = 4 //RandStringRunes(8)

				// perform mutation operation here
				// for e.g. create a Todo and save to DB.
				newTodo := Todo{
					Id:   newID,
					Text: Text,
					Done: false,
				}

				TodoList = append(TodoList, newTodo)

				// return the new Todo object that we supposedly save to DB
				// Note here that
				// - we are returning a `Todo` struct instance here
				// - we previously specified the return Type to be `todoType`
				// - `Todo` struct maps to `todoType`, as defined in `todoType` ObjectConfig`
				return newTodo, nil
			},
		},
		/*
			curl -g 'http://localhost:8081/graphql?query=mutation+_{updateTodo(Id:"a",Done:true){Id,Text,Done}}'
		*/
		"updateTodo": &graphql.Field{
			Type:        todoType, // the return type for this field
			Description: "Update existing todo, mark it Done or not Done",
			Args: graphql.FieldConfigArgument{
				"Done": &graphql.ArgumentConfig{
					Type: graphql.Boolean,
				},
				"Id": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.Int),
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				// marshall and cast the argument value
				DoneParam, _ := params.Args["Done"].(bool)
				IdParam, _ := params.Args["Id"].(int64)

				todo := Todo{Id: IdParam, Done: DoneParam}

				engine, _ = xorm.NewEngine("sqlite3", "./test.db")

				affected, err := engine.Id(IdParam).Update(todo)
				fmt.Println(affected)
				//_, err := engine.Update(&todo)
				if err != nil {
					fmt.Println(err)
					return nil, err
				}

				engine.Get(&todo)

				return todo, err
			},
		},
	},
})

// root query
// we just define a trivial example here, since root query is required.
// Test with curl
// curl -g 'http://localhost:8081/graphql?query={lastTodo{Id,Text,Done}}'
var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{

		/*
		   curl -g 'http://localhost:8081/graphql?query={todo(Id:"b"){Id,Text,Done}}'
		*/
		"todo": &graphql.Field{
			Type:        todoType,
			Description: "Get single todo",
			Args: graphql.FieldConfigArgument{
				"Id": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {

				idQuery, isOK := params.Args["Id"].(int64)
				if isOK {

					engine, _ = xorm.NewEngine("sqlite3", "./test.db")

					var todo = Todo{Id: idQuery}
					_, err := engine.Get(&todo)
					return todo, err
				}

				return Todo{}, nil
			},
		},

		/*
		   curl -g 'http://localhost:8081/graphql?query={todoList{Id,Text,Done}}'
		*/
		"todoList": &graphql.Field{
			Type:        graphql.NewList(todoType),
			Description: "List of todos",
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {

				var all []Todo

				engine, _ = xorm.NewEngine("sqlite3", "./test.db")

				_ = engine.Find(&all)
				fmt.Println("%v", all)
				return TodoList, nil
			},
		},
	},
})

// define schema, with our rootQuery and rootMutation
var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query:    rootQuery,
	Mutation: rootMutation,
})

func serveGraphQL(s graphql.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sendError := func(err error) {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}

		req := &graphiql.Request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			sendError(err)
			return
		}

		res := graphql.Do(graphql.Params{
			Schema:        s,
			RequestString: req.Query,
		})

		if err := json.NewEncoder(w).Encode(res); err != nil {
			sendError(err)
		}
	}
}

func main() {

	engine, err := xorm.NewEngine("sqlite3", "./test.db")

	if err != nil {
		fmt.Println(err)
		return
	}

	err = engine.Sync2(new(Todo))

	engine.Insert(TodoList)

	todo := &Todo{}
	engine.Id(1).Get(todo)

	todo.Done = true
	engine.Update(todo)

	engine.Id(1).Get(todo)

	//engine.Insert(&Account{Id: 1, Name: "sdfsdf", Balance: 12.0})

	// a := &Account{}
	// _, _ = engine.Id(1).Get(a)

	// a.Balance += 10
	// engine.Update(a)

	// engine.Id(1).Get(a)

	http.HandleFunc("/", graphiql.ServeGraphiQL)
	http.HandleFunc("/graphql", serveGraphQL(schema))

	fmt.Println("Now server is running on port 8081")
	fmt.Println("Get single todo: curl -g 'http://localhost:8081/graphql?query={todo(id:\"b\"){id,text,done}}'")
	fmt.Println("Create new todo: curl -g 'http://localhost:8081/graphql?query=mutation+_{createTodo(text:\"My+new+todo\"){id,text,done}}'")
	fmt.Println("Update todo: curl -g 'http://localhost:8081/graphql?query=mutation+_{updateTodo(id:\"a\",done:true){id,text,done}}'")
	fmt.Println("Load todo list: curl -g 'http://localhost:8081/graphql?query={todoList{id,text,done}}'")
	fmt.Println("Access the web app via browser at 'http://localhost:8081'")

	http.ListenAndServe(":8081", nil)

}
