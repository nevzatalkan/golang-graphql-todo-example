# Go GraphQL ToDo example

An example that consists of basic core GraphQL queries and mutations.

Execute queries via shell.

```
// To get single ToDo item by ID
curl -g 'http://localhost:8080/graphql?query={todo(id:"b"){id,text,done}}'

// To create a ToDo item
curl -g 'http://localhost:8080/graphql?query=mutation+_{createTodo(text:"My+new+todo"){id,text,done}}'

// To get a list of ToDo items
curl -g 'http://localhost:8080/graphql?query={todoList{id,text,done}}'

// To update a ToDo
curl -g 'http://localhost:8080/graphql?query=mutation+_{updateTodo(id:"b",text:"My+new+todo+updated",done:true){id,text,done}}'
```

## Web App

Access the web app at `http://localhost:8080/`.
