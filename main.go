package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	gql "github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

type CustomID struct {
	value string
}

func (id *CustomID) String() string {
	return id.value
}

func NewCustomID(v string) *CustomID {
	return &CustomID{value: v}
}

var CustomScalarType = gql.NewScalar(gql.ScalarConfig{
	Name:        "CustomScalarType",
	Description: "The `CustomScalarType` scalar type represents an ID Object.",
	// Serialize serializes `CustomID` to string.
	Serialize: func(value interface{}) interface{} {
		switch value := value.(type) {
		case CustomID:
			return value.String()
		case *CustomID:
			v := *value
			return v.String()
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `CustomID`.
	ParseValue: func(value interface{}) interface{} {
		switch value := value.(type) {
		case string:
			return NewCustomID(value)
		case *string:
			return NewCustomID(*value)
		default:
			return nil
		}
	},
	// ParseLiteral parses GraphQL AST value to `CustomID`.
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return NewCustomID(valueAST.Value)
		default:
			return nil
		}
	},
})

type Student struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"des,omitempty"`
	Score       float64  `json:"price"`
	NationalID  CustomID `json:"nationalid"`
}

var students []Student

var studentType = gql.NewObject(
	gql.ObjectConfig{
		Name: "Student",
		Fields: gql.Fields{
			"id": &gql.Field{
				Type: gql.Int,
			},
			"name": &gql.Field{
				Type: gql.String,
			},
			"des": &gql.Field{
				Type: gql.String,
			},
			"score": &gql.Field{
				Type: gql.Float,
			},
			"nationalid": &gql.Field{
				Type: CustomScalarType,
			},
		},
	},
)

// http://localhost:8080/student?query={student(id:1){name,des,score,nationalid}}
func GetStudent(p gql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, errors.New("invalid id")
	}
	for _, s := range students {
		if int(s.ID) == id {
			return s, nil
		}
	}
	return nil, errors.New("student not found")
}

// http://localhost:8080/student?query={list{id,name,des,score,nationalid}}
func ListStudents(params gql.ResolveParams) (interface{}, error) {
	return students, nil
}

var queryType = gql.NewObject(
	gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"student": &gql.Field{
				Type:        studentType,
				Description: "Get student by id",
				Args: gql.FieldConfigArgument{
					"id": &gql.ArgumentConfig{
						Type: gql.NewNonNull(gql.Int),
					},
				},
				Resolve: GetStudent,
			},
			"list": &gql.Field{
				Type:        gql.NewList(studentType),
				Description: "Get students list",
				Resolve:     ListStudents,
			},
		},
	})

//http://localhost:8080/student?query=mutation+_{enroll(name:"John Doe",des:"John Doe is an excellent student",score:100.00,){id,name,des,score,nationalid}}
func EnrollStudent(params gql.ResolveParams) (interface{}, error) {
	rand.Seed(time.Now().UnixNano())
	student := Student{
		ID:          int64(rand.Intn(100000)), // generate random ID
		Name:        params.Args["name"].(string),
		Description: params.Args["des"].(string),
		Score:       params.Args["score"].(float64),
	}
	students = append(students, student)
	return student, nil
}

// http://localhost:8080/student?query=mutation+_{update(id:1,score:3.95){id,name,des,score,nationalid}}
func UpdateStudent(params gql.ResolveParams) (interface{}, error) {
	id, _ := params.Args["id"].(int)
	student := Student{}
	for i, s := range students {
		if int64(id) == s.ID {
			name, ok := params.Args["name"].(string)
			if ok {
				students[i].Name = name
			}
			des, ok := params.Args["des"].(string)
			if ok {
				students[i].Description = des
			}
			score, ok := params.Args["score"].(float64)
			if ok {
				students[i].Score = score
			}
			student = students[i]
			break
		}
	}
	return student, nil
}

// http://localhost:8080/student?query=mutation+_{leave(id:1){id,name,des,score,nationalid}}
func StudentLeave(params gql.ResolveParams) (interface{}, error) {
	id, _ := params.Args["id"].(int)
	student := Student{}
	for i, s := range students {
		if int64(id) == s.ID {
			student = students[i]
			students = append(students[:i], students[i+1:]...)
		}
	}
	return student, nil
}

var mutationType = gql.NewObject(gql.ObjectConfig{
	Name: "Mutation",
	Fields: gql.Fields{
		"enroll": &gql.Field{
			Type:        studentType,
			Description: "Enroll new student",
			Args: gql.FieldConfigArgument{
				"name": &gql.ArgumentConfig{
					Type: gql.NewNonNull(gql.String),
				},
				"des": &gql.ArgumentConfig{
					Type: gql.String,
				},
				"score": &gql.ArgumentConfig{
					Type: gql.Float,
				},
			},
			Resolve: EnrollStudent,
		},

		"update": &gql.Field{
			Type:        studentType,
			Description: "Update student by id",
			Args: gql.FieldConfigArgument{
				"id": &gql.ArgumentConfig{
					Type: gql.NewNonNull(gql.Int),
				},
				"name": &gql.ArgumentConfig{
					Type: gql.String,
				},
				"des": &gql.ArgumentConfig{
					Type: gql.String,
				},
				"score": &gql.ArgumentConfig{
					Type: gql.Float,
				},
			},
			Resolve: UpdateStudent,
		},

		"leave": &gql.Field{
			Type:        studentType,
			Description: "Student leave by id",
			Args: gql.FieldConfigArgument{
				"id": &gql.ArgumentConfig{
					Type: gql.NewNonNull(gql.Int),
				},
			},
			Resolve: StudentLeave,
		},
	},
})

var schema, _ = gql.NewSchema(
	gql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	},
)

func execute(query string, schema gql.Schema) *gql.Result {
	result := gql.Do(gql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		fmt.Printf("failed with errors: %v", result.Errors)
		return &gql.Result{}
	}
	return result
}

func initStudents(s *[]Student) {
	s1 := Student{ID: 1, Name: "Alice", Description: "Alice is a diligent student", Score: 4, NationalID: *NewCustomID("asdfsdfs")}
	s2 := Student{ID: 2, Name: "Bob", Description: "Bob is naughty", Score: 3.0, NationalID: *NewCustomID("a23fi43gg")}
	*s = append(*s, s1, s2)
}

func main() {
	initStudents(&students)

	http.HandleFunc("/student", func(w http.ResponseWriter, r *http.Request) {
		result := execute(r.URL.Query().Get("query"), schema)
		json.NewEncoder(w).Encode(result)
		log.Println("new request: ", r.URL.Query().Get("query"))
		log.Println("result: ", result)
	})

	fmt.Println("GraphQL Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}
