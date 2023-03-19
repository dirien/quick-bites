# Rust Development: Creating a REST API with Actix Web for Beginners

## Introduction

In this blog article, I want to show you how to build a REST API in Rust using Actix Web. And what is the best way to learn something new? By trying it out yourself, learn from your mistakes and improve your skills. We are going to implement the REST API for the famous TODO app.

The API will have the following endpoints:

* Create a new TODO item

* Get a TODO item by its unique identifier

* Delete a TODO item by its unique identifier

* Get all TODO items

* Update an existing TODO item


But before we start implementing the API, let us talk about REST APIs in general to get a better understanding of what we are going to build.

## What is a REST API?

REST is an acronym for REpresentational State Transfer. The term REST was first introduced by [Roy Fielding](https://en.wikipedia.org/wiki/Roy_Fielding) in his doctoral dissertation "**Architectural Styles and the Design of Network-based Software Architectures**". REST is an architectural style for building distributed systems, which are often web services. REST is not a standard, thus it does not enforce any rules on how to build a REST API. But there are some high-level guidelines that you should follow. REST-based systems interact with each other using HTTP (HyperText Transfer Protocol).

RESTful systems consist of two parts: The client and the server. The client is the system that initiates the request for a resource and the server has the resource and sends it back to the client.

### Architectural Constraints of a REST API:

There are six (one of them is optional) architectural constraints that you should follow when building a REST API:

* Uniform Interface

* Stateless

* Cacheable

* Client-Server

* Layered System

* Code on Demand (this one is optional)


Let us take a look at each of them in more detail.

#### Uniform Interface

The uniform interface constraint states that every REST API should have a uniform interface and this distinguishes it from non-RESTful APIs. A uniform interface means that there must be a way to interact with server resources independent of the client device or type of application.

To adhere to this constraint, Fielding defined four properties that every REST API should follow:

* Identification of resources

* Manipulation of resources through representations

* Self-descriptive messages

* Hypermedia as the engine of application state (HATEOAS)


This means in practice:

* You should use nouns instead of verbs in your resource names. Example: /todos instead of /getTodos

* The use of HTTP methods like GET, POST, PUT, and DELETE to identify the operation that clients want to perform on the resource.

* We should use always the plural form of the resource name. Example: `/todos` instead of `/todo`

* Always send a proper HTTP status code to indicate the success or failure of the request. Example: `200` for indicating success or `404` for indicating that the resource was not found.


#### Stateless

The stateless constraint is that a server should not store any context on the server. All the necessary states to handle a request part of the request itself. Statelessness is a great way to scale our system and increase its availability.

#### Cacheable

A good REST API should be cacheable to eliminate unnecessary network traffic. In some cases, there are chances that the user might receive stale data.

#### Client-Server

This constraint states that the client and the server should follow a strict separation of concerns. So every application can evolve independently without any dependency on the other.

#### Layered System

With this constraint, we can use a layered system architecture to build our REST API. We can deploy the API on one server while storing the data on another server and use another server to handle any authentication requests.

#### Code on Demand (Optional)

This optional constraint states that the server can send executable code to the client. I honestly never used this and have no idea when this would be useful.

## What is Actix Web?

Actix Web is a very popular web framework for Rust. Actix Web was once built on top of the Actix actor framework, but now it is unrelated to the Actix actor framework. An application built with the Actix Web framework will expose an HTTP server contained within a native Rust binary.

%[https://github.com/actix/actix-web] 

## Prerequisites

Before we start, we need to make sure we have the following tools installed:

* [Rust](https://www.rust-lang.org)

* An IDE or text editor of your choice


## Initialize the project

```shell
cargo init
```

We will add the `actix-web`, `serde`, `chrono` and `uuid` crates to our project.

```shell
cargo add actix-web
cargo add serde --features derive
cargo add chrono --features serde
cargo add uuid --features v4
```

## Creating the application entry point

Now with the project initialized, we can start implementing the REST API. Add the following code to the `main.rs` file:

```rust
use actix_web::{get, web, App, HttpResponse, HttpServer, Responder, Result};
use serde::{Serialize};

#[derive(Serialize)]
pub struct Response {
    pub message: String,
}

#[get("/health")]
async fn healthcheck() -> impl Responder {
    let response = Response {
        message: "Everything is working fine".to_string(),
    };
    HttpResponse::Ok().json(response)
}


async fn not_found() -> Result<HttpResponse> {
    let response = Response {
        message: "Resource not found".to_string(),
    };
    Ok(HttpResponse::NotFound().json(response))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| App::new().service(healthcheck).default_service(web::route().to(not_found)))
        .bind(("127.0.0.1", 8080))?
        .run()
        .await
}
```

The code is doing the following things:

* Defines a struct `Response` which will be used to send a response to the client.

* Creates a handler function `health`, which can late be probed by any client to check if the server is up and running.

* Uses the `#[actix_web::main]` macro to run the `main` function as an asynchronous function with the `actix-web` runtime. The `main` function does the following:

    * Creates a new server using the `HttpServer` struct. The `HttpServer` struct uses a closure to server any incoming requests using the `App` instance. The `App` instance is used to register all the routes that the server should handle.

    * Register a default handler function `not_found` which will be called if the client requests a resource that is not registered with the server.

    * Configures the server to listen on `localhost:8080` and starts the server.


When we run the application, we should see the following output:

```bash
cargo run -q
```

And test the server by sending a request to the `/health` endpoint:

```bash
curl localhost:8080/health -vvv
*   Trying 127.0.0.1:8080...
* Connected to localhost (127.0.0.1) port 8080 (#0)
> GET /health HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.79.1
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 59
< content-type: application/json
< date: Sat, 11 Mar 2023 08:27:37 GMT
< 
* Connection #0 to host localhost left intact
{"status":"success","message":"Everything is working fine"}
```

## Organizing the code with modules

In Rust, we can use modules to hierarchically split our code into different logical units and manage the visibility between them. This helps us to keep the code we write clean and organized.

In our project, we will create under the `src` directory following three folders: `api`, `models` and `repository`. We add a `mod.rs` file to each of these folders. The `mod.rs` file is used to define the modules and the visibility as per default, all the items in the module are private.

```bash
mkdir src/api src/models src/repository
touch src/api/mod.rs src/models/mod.rs src/repository/mod.rs
```

Now we can add the reference to the `mod.rs` files in the `main.rs` file:

```rust
use actix_web::{get, web, App, HttpResponse, HttpServer, Responder, Result};
use serde::{Serialize};

mod api;
mod models;
mod repository;


#[derive(Serialize)]
pub struct Response {
    pub status: String,
    pub message: String,
}
// ...
```

## Creating the first API endpoint

First, we are going to create a model for the `Todo` resource. Add the following code to the `src/models/todo.rs` file:

```rust
use chrono::prelude::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Todo {
    pub id: Option<String>,
    pub title: String,
    pub description: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}
```

This `Todo` rust struct is used to represent our `Todo` resource. The use of the `derive` macro generates the implementation support for formatting, cloning, serialization and deserialization of the struct.

The `pub` modifier makes the fields of the struct public, which means that they can be accessed from other files and modules.

As the last step, we must register the `todo.rs` file as part of the `models` module. Add the following line to the `src/models/mod.rs` file:

```rust
pub mod todo;
```

With the creation of the model out of the way, we can now create our database logic. I want to create in the future a dedicated article about how to use a database with `actix-web`. For now, we will use a simple in-memory hashmap to store our data and use mutexes to make the data thread-safe.

Create a new file `src/repository/database.rs` and add the following code:

```rust
use std::fmt::Error;
use chrono::prelude::*;
use std::sync::{Arc, Mutex};

use crate::models::todo::Todo;

pub struct Database {
    pub todos: Arc<Mutex<Vec<Todo>>>,
}

impl Database {
    pub fn new() -> Self {
        let todos = Arc::new(Mutex::new(vec![]));
        Database { todos }
    }

    pub fn create_todo(&self, todo: Todo) -> Result<Todo, Error> {
        let mut todos = self.todos.lock().unwrap();
        let id = uuid::Uuid::new_v4().to_string();
        let created_at = Utc::now();
        let updated_at = Utc::now();
        let todo = Todo {
            id: Some(id),
            created_at: Some(created_at),
            updated_at: Some(updated_at),
            ..todo
        };
        todos.push(todo.clone());
        Ok(todo)
    }
}
```

The code for our `mock` database is doing the following things:

* Defines a struct called `Database` which contains a `todos` field of type `Arc<Mutex<Vec<Todo>>>`. The `Arc` struct is used to create a thread-safe reference-counting pointer. The `Mutex` struct is used to create a mutual exclusion primitive. The `Mutex` is used to make sure that only one thread can access the data at a time.

* Implements a `new` function which creates a new instance of the `Database` struct. We're going to wrap the `todos` field in an `Arc` and `Mutex` to make it thread-safe and return the `Database` struct.

* The last piece is our `create_todo` function which takes a `Todo` struct as an argument and returns a `Result` type. The `create_todo` function does the following:

    * Locks the `todos` field using the `lock` method of the `Mutex` struct. This will guarantee that only one thread can access the data at a time.

    * Generates a new `id` using `uuid`.

    * Fill the `created_at` and `updated_at` timestamp using the `chrono`.

    * Creates a new `Todo` struct by cloning the `todo` argument and set the `id`, `created_at` and `updated_at` fields with values.

    * Adds the new `Todo` struct to the `todos` vector.

    * And finally returns the new `Todo` struct to the caller.


That's a lot of code, but we are almost done! Now we need to create our `create_todo` API endpoint. Create a new file called `src/api/api.rs` and add the following code:

```rust
use actix_web::web;
use actix_web::{web::{
    Data,
    Json,
}, post, HttpResponse};
use crate::{models::todo::Todo, repository::database::Database};


#[post("/todos")]
pub async fn create_todo(db: Data<Database>, new_todo: Json<Todo>) -> HttpResponse {
    let todo = db.create_todo(new_todo.into_inner());
    match todo {
        Ok(todo) => HttpResponse::Ok().json(todo),
        Err(err) => HttpResponse::InternalServerError().body(err.to_string()),
    }
}


pub fn config(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/api")
            .service(create_todo)
    );
}
```

The above code is doing this:

* Creates a `create_todo` function which takes a `Data<Database>` and `Json<Todo>` as arguments. Then it calls the `create_todo` function of the `Database` struct and returns the result to the caller. As we return a `Result` type from the `create_todo` function, we can use the `match` statement to handle the success and error cases.

* Creates a `config` function which takes a `&mut web::ServiceConfig` as an argument. In this function, we are going to register all our API endpoints under the `/api` path by using the `web::scope` method.


The last step before we can run our application is to wire everything together. Open the `src/main.rs` file and add the following code to the `main` function:

```rust
#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let todo_db = repository::database::Database::new();
    let app_data = web::Data::new(todo_db);

    HttpServer::new(move ||
        App::new()
            .app_data(app_data.clone())
            .configure(api::api::config)
            .service(healthcheck)
            .default_service(web::route().to(not_found))
            .wrap(actix_web::middleware::Logger::default())
    )
        .bind(("127.0.0.1", 8080))?
        .run()
        .await
}
```

This will create a new instance of the `Database` struct and register it as a `web::Data` struct. Additionally, we set the configuration for our API endpoints via the `configure` method of the `App` struct.

Now we can run our application by executing the following command:

```bash
cargo run
```

And try to create a new `Todo` by executing the following command:

```bash
curl -X POST -H "Content-Type: application/json" -d '{"title": "My first todo", "description": "This is my first todo"}' http://localhost:8080/api/todos
```

If everything worked as expected, you should see the following output:

```json
{
  "id": "d70053a9-721d-4c20-9a27-b26b4fbaecae",
  "title": "My first todo",
  "description": "This is my first todo",
  "created_at": "2023-03-11T10:33:56.441332Z",
  "updated_at": "2023-03-11T10:33:56.441390Z"
}
```

Now we can continue to implement the remaining API endpoints.

## Implementing the remaining API endpoints

### GET `/todos`

The next endpoint, we are going to implement is to get all `Todo` items. Open the `src/api/api.rs` file and add this new function:

```rust
#[get("/todos")]
pub async fn get_todos(db: web::Data<Database>) -> HttpResponse {
    let todos = db.get_todos();
    HttpResponse::Ok().json(todos)
}
```

Add the `get_todos` function to the `config` function:

```rust
pub fn config(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/api")
            .service(create_todo)
            .service(get_todos)
    );
}
```

Finally, we need to implement the `get_todos` function in the `src/repository/database.rs` file:

```rust
impl Database {
    // ...    

    pub fn get_todos(&self) -> Vec<Todo> {
        let todos = self.todos.lock().unwrap();
        todos.clone()
    }
}
```

### GET `/todos/{id}`

Now we are going to implement is to get a single `Todo` item by its `id`. Open the `src/api/api.rs` file and add this code snippet:

```rust
#[get("/todos/{id}")]
pub async fn get_todo_by_id(db: web::Data<Database>, id: web::Path<String>) -> HttpResponse {
    let todo = db.get_todo_by_id(&id);
    match todo {
        Some(todo) => HttpResponse::Ok().json(todo),
        None => HttpResponse::NotFound().body("Todo not found"),
    }
}
```

Again, add the `get_todo_by_id` function to the `config` function:

```rust
pub fn config(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/api")
            .service(create_todo)
            .service(get_todos)
            .service(get_todo_by_id)
    );
}
```

Now we need to implement the `get_todo_by_id` function in the `src/repository/database.rs` file:

```rust
impl Database {

    // ...    

    pub fn get_todo_by_id(&self, id: &str) -> Option<Todo> {
        let todos = self.todos.lock().unwrap();
        todos.iter().find(|todo| todo.id == Some(id.to_string())).cloned()
    }
}
```

### PUT `/todos/{id}`

Now it's time to implement the `PUT` endpoint to update a `Todo` item. Open the `src/api/api.rs` file and add this code snippet:

```rust
#[put("/todos/{id}")]
pub async fn update_todo_by_id(db: web::Data<Database>, id: web::Path<String>, updated_todo: web::Json<Todo>) -> HttpResponse {
    let todo = db.update_todo_by_id(&id, updated_todo.into_inner());
    match todo {
        Some(todo) => HttpResponse::Ok().json(todo),
        None => HttpResponse::NotFound().body("Todo not found"),
    }
}
```

Add the `update_todo_by_id` function to the `config` function:

```rust
pub fn config(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/api")
            .service(create_todo)
            .service(get_todos)
            .service(get_todo_by_id)
            .service(update_todo_by_id)
    );
}
```

And add the `update_todo_by_id` function to the `src/repository/database.rs` file:

```rust
impl Database {

    // ...    

    pub fn update_todo_by_id(&self, id: &str, todo: Todo) -> Option<Todo> {
        let mut todos = self.todos.lock().unwrap();
        let updated_at = Utc::now();
        let todo = Todo {
            id: Some(id.to_string()),
            updated_at: Some(updated_at),
            ..todo
        };
        let index = todos.iter().position(|todo| todo.id == Some(id.to_string()))?;
        todos[index] = todo.clone();
        Some(todo)
    }
}
```

### DELETE `/todos/{id}`

Last but not least, we are going to implement the `DELETE` endpoint to delete a `Todo` item. Open the `src/api/api.rs` file and add this code snippet:

```rust
#[delete("/todos/{id}")]
pub async fn delete_todo_by_id(db: web::Data<Database>, id: web::Path<String>) -> HttpResponse {
    let todo = db.delete_todo_by_id(&id);
    match todo {
        Some(todo) => HttpResponse::Ok().json(todo),
        None => HttpResponse::NotFound().body("Todo not found"),
    }
}
```

Add this function to the `config` function:

```rust
pub fn config(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/api")
            .service(create_todo)
            .service(get_todos)
            .service(get_todo_by_id)
            .service(update_todo_by_id)
            .service(delete_todo_by_id)
    );
}
```

And add the `delete_todo_by_id` function to the `src/repository/database.rs` file:

```rust
impl Database {

    // ...    

    pub fn delete_todo_by_id(&self, id: &str) -> Option<Todo> {
        let mut todos = self.todos.lock().unwrap();
        let index = todos.iter().position(|todo| todo.id == Some(id.to_string()))?;
        Some(todos.remove(index))
    }
}
```

We are done! Now we can start the server and test our API!

## Testing the API

Start the server:

```bash
cargo run
```

And execute the following `curl` commands to test the API:

```bash
# Create a new Todo items
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy milk", "description": "Buy 2 liters of milk"}' http://localhost:8080/api/todos
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy eggs", "description": "Buy 12 eggs"}' http://localhost:8080/api/todos
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy bread", "description": "Buy 1 loaf of bread"}' http://localhost:8080/api/todos

# Get all Todo items
curl -s http://localhost:8080/api/todos | jq  
[
  {
    "id": "590538de-56c4-4057-b4e6-c91021fc04be",
    "title": "Buy milk",
    "description": "Buy 2 liters of milk",
    "created_at": "2023-03-11T11:58:28.176321Z",
    "updated_at": "2023-03-11T11:58:28.176376Z"
  },
  {
    "id": "54f7695f-55a0-423f-9aba-0d2ec323eef3",
    "title": "Buy eggs",
    "description": "Buy 12 eggs",
    "created_at": "2023-03-11T11:58:28.183312Z",
    "updated_at": "2023-03-11T11:58:28.183314Z"
  },
  {
    "id": "cd574ca3-0d18-4e34-adad-c493140607a5",
    "title": "Buy bread",
    "description": "Buy 1 loaf of bread",
    "created_at": "2023-03-11T11:58:28.189685Z",
    "updated_at": "2023-03-11T11:58:28.189687Z"
  }
]


# Get a Todo item by id
curl -s http://localhost:8080/api/todos/590538de-56c4-4057-b4e6-c91021fc04be | jq
{
  "id": "590538de-56c4-4057-b4e6-c91021fc04be",
  "title": "Buy milk",
  "description": "Buy 2 liters of milk",
  "created_at": "2023-03-11T11:58:28.176321Z",
  "updated_at": "2023-03-11T11:58:28.176376Z"
}


# Update a Todo item by id
curl -s -X PUT -H "Content-Type: application/json" -d '{"title": "Buy 2 liters of milk", "description": "Buy 2 liters of milk"}' http://localhost:8080/api/todos/590538de-56c4-4057-b4e6-c91021fc04be | jq
{
  "id": "590538de-56c4-4057-b4e6-c91021fc04be",
  "title": "Buy 2 liters of milk",
  "description": "Buy 2 liters of milk",
  "created_at": "2023-03-11T11:58:28.176321Z",
  "updated_at": "2023-03-11T11:58:28.176376Z"
}

# Delete a Todo item by id
curl -s -X DELETE http://localhost:8080/api/todos/590538de-56c4-4057-b4e6-c91021fc04be | jq
{
  "id": "590538de-56c4-4057-b4e6-c91021fc04be",
  "title": "Buy 2 liters of milk",
  "description": "Buy 2 liters of milk",
  "created_at": "2023-03-11T11:58:28.176321Z",
  "updated_at": "2023-03-11T11:58:28.176376Z"
}

# Get all Todo items
curl -s http://localhost:8080/api/todos | jq
[
  {
    "id": "54f7695f-55a0-423f-9aba-0d2ec323eef3",
    "title": "Buy eggs",
    "description": "Buy 12 eggs",
    "created_at": "2023-03-11T11:58:28.183312Z",
    "updated_at": "2023-03-11T11:58:28.183314Z"
  },
  {
    "id": "cd574ca3-0d18-4e34-adad-c493140607a5",
    "title": "Buy bread",
    "description": "Buy 1 loaf of bread",
    "created_at": "2023-03-11T11:58:28.189685Z",
    "updated_at": "2023-03-11T11:58:28.189687Z"
  }
]
```

## Conclusion

Congratulations! If you made it this far, you have successfully created a RESTful API with Rust and `artix-web`. You learned some basic concepts of RESTful APIs and how a possible implementation could look like. You also learned to set up an in-memory database using `Mutex` and `Arc`.

In the next blog post, we will take a look on how to replace the in-memory database with a real database. Tell me in the comments what database you would like me to use and why it should be `postgres`.

## Resources

* https://restfulapi.net/rest-architectural-constraints/

* https://restfulapi.net/

* https://www.techtarget.com/searchapparchitecture/tip/The-5-essential-HTTP-methods-in-RESTful-API-development

* https://medium.com/@andreasreiser94/why-hateoas-is-useless-and-what-that-means-for-rest-a65194471bc8

* https://actix.rs/

* https://hackernoon.com/easily-understand-rust-modules-across-multiple-files-with-this-guide

* https://aeshirey.github.io/code/2020/12/23/arc-mutex-in-rust.html
