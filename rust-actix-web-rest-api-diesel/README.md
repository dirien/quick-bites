# Building a RESTful API with Actix Web and Diesel for Persistent Data Storage

## Introduction

In my last blog article, we created a REST API with Actix Web with in-memory persistence. This was a great way to get
started with [Actix Web](https://github.com/actix/actix-web), but it's not very useful in the real world scenario. For
this blog article, we will finally add persistence to our demo application. We will use PostgreSQL as our database and
Diesel.rs to interact with it.

If you want to follow up with the last blog article, you can find re-read it here:

<Link to last blog article>

## What is Diesel?

Diesel is a powerful and efficient Object-Relational Mapping (ORM) framework for Rust programming language. It provides
a type-safe and composable query API that allows to interact with databases in a safe way.

Diesel supports various databases like PostgreSQL, MySQL and SQLite. It provides a rich set of features including
migrations, database schema management, and support for complex queries. Diesel's compile-time checking ensures that
we catch errors early in the development process and reduces the potential of runtime errors.

For more information about Diesel, please visit the official repository or the documentation:

https://github.com/diesel-rs/diesel

## Prerequisites

* [Rust](https://www.rust-lang.org)

* An IDE or text editor of your choice

* [Docker](https://www.docker.com) and `docker-compose` installed

* optional: If you want to interact with the PostgreSQL database, you can install psql with `brew install postgresql`

* The code from the last blog article, which you can find here: <Link to last blog article>

## Setting up the Database

For this demo application we will run the PostgreSQL database in a Docker container. In a production environment, you
would use probably a managed database service like AWS RDS or Azure Database for PostgreSQL.

Create a `postgres.yaml` file in the root of your project and add the following content:

```yaml
version: '3.8'

services:
  db:
    container_name: postgres
    image: postgres:14.7-alpine
    restart: always
    environment:
      POSTGRES_USER: superuser
      POSTGRES_PASSWORD: superpassword
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
volumes:
  postgres-data:
```

This Docker Compose file will create a PostgreSQL database and make it available on port 5432. Don't mind the
credentials, they are only for demo purposes.

Now, we can start the database with following command:

```bash
docker-compose -f postgres.yaml up -d
```

> **Note**: You can use the `-d` flag to run the database in the background.

You should see the following output:

```bash
Creating network "rust-actix-web-rest-api-diesel_default" with the default driver
Creating volume "rust-actix-web-rest-api-diesel_postgres-data" with default driverloud       docker-index                   docutils                                                                                                                                           
Pulling db (postgres:14.7-alpine)...
15.2-alpine: Pulling from library/postgres
af6eaf76a39c: Already exists
71286d2ce0cc: Pull complete
b82afe47906a: Pull complete
75d514bb4aa7: Pull complete
217da6f41d9e: Pull complete
39a3f4823126: Pull complete
ed6571a6afcc: Pull complete
8ae7d38f54c4: Pull complete
Digest: sha256:1f86ede0903f60ecd2eb630b15803567324da7aa0d1f7bbc3a8f1fe5247a4592
Status: Downloaded newer image for postgres:14.7-alpine
Creating postgres ... done
Attaching to postgres
postgres | The files belonging to this database system will be owned by user "postgres".
postgres | This user must also own the server process.
postgres | 
...
postgres | 2023-03-18 10:08:51.522 UTC [52] LOG:  database system was shut down at 2023-03-18 10:08:51 UTC
postgres | 2023-03-18 10:08:51.526 UTC [1] LOG:  database system is ready to accept connections
```

You can now connect to the database with the following command:

```bash
psql postgresql://superuser:superpassword@localhost:5432
# or 
psql -h localhost -p 5432 -U superuser
```

## Setting up the Diesel CLI

The Diesel CLI is a command-line tool that allows us to interact with the database. To install it, run the following
command:

```bash
cargo install diesel_cli --no-default-features --features postgres
```

> **Note**: You man need to have `libpq` and `postgresql` installed on your machine. On macOS, you can install them
> with `brew install libpq postgresql`.

Now you can create a `.env` file in the root of your project and add the following content:

```bash
DATABASE_URL=postgresql://superuser:superpassword@localhost:5432/todo
```

And run the diesel setup command. This will create our database (if it doesn't exist yet) and create the migrations
directory.

```bash
diesel setup
```

You should see the following output:

```bash
Creating migrations directory at: /Users/dirien/Tools/repos/quick-bites/rust-actix-web-rest-api-diesel/migrations
Creating database: todo 
```

Now, we can create our Todo table to store our todos. We do this with a migration. Migrations allow us to evolve our
schema. Each new migration can be applied (up.sql) or reverted (down.sql).

```bash
diesel migration generate create_todo_table
```

You should see the following output:

```bash
Creating migrations/2023-03-18-103853_create_todo_table/up.sql
Creating migrations/2023-03-18-103853_create_todo_table/down.sql
```

Now, we can add the following SQL to the `up.sql` file to create our Todo table. See the `src/models/todo.rs` file for
the struct definition, we used in the last blog article.

```sql
CREATE TABLE todos
(
    id          VARCHAR(255) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMP,
    updated_at  TIMESTAMP
)
```

Run the following command to apply the migration:

```bash
diesel migration run
```

This will create the `todos` table in our database and generate a `schema.rs` file in the `src` directory.

To roll back the migration, we need to add the following SQL to the `down.sql` file:

```sql
DROP TABLE todos
```

And run the following command:

```bash
diesel migration redo
```

As you see, diesel is generating the schema file for us, but we want to change the location of the file. To do this, we
head over to the `diesel.toml` file and change `print_schema` to the following:

```toml
# omitting the rest of the file
[print_schema]
file = "src/models/schema.rs"
# omitting the rest of the file
```

My `schema.rs` file looks like this:

```rust
// @generated automatically by Diesel CLI.

diesel::table! {
    todos (id) {
        id -> Varchar,
        title -> Varchar,
        description -> Nullable<Text>,
        created_at -> Nullable<Timestamp>,
        updated_at -> Nullable<Timestamp>,
    }
}
```

Notable is the `table!` macro. This macro creates a lot of code for us based on the database schema. You will see later
on how exactly this works.

## Connecting to the Database

Before we can start to set up our database connection, we need to add the following dependencies to our Rust project:

```bash
cargo add dotenv
```

And add the following to the `Cargo.toml` file:

```toml
[dependencies]
diesel = { version = "2.0.3", features = ["postgres", "r2d2", "chrono", "uuid"] }
```

In the last blog article, we already created an abstraction layer for our database connection. That means, we need to do
any changes to support the new database connection in the `database` implementation and add some diesel macros to
our `Todo` struct.

Head over to the `src/models/todo.rs` file and add the following imports and new traits to the `Todo` struct:

```rust
use serde::{Deserialize, Serialize};
use diesel::{Queryable, Insertable, AsChangeset};

#[derive(Serialize, Deserialize, Debug, Clone, Queryable, Insertable, AsChangeset)]
#[diesel(table_name = crate::repository::schema::todos)]
pub struct Todo {
    #[serde(default)]
    pub id: String,
    pub title: String,
    pub description: Option<String>,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
}
```

The `Queryable` trait allows us to load a Todo from the database while the `Insertable` trait is for inserting a Todo
and the `AsChangeset` to update a Todo.

You may have spot the `#[serde(default)]` attribute on the `id` field. This is because we want to have a default value
and avoid to have a nullable field as primary key.

Now we need to change the `database` implementation to support the new database connection. Open
the `src/repository/database.rs` file and add the following imports:

```rust
use chrono::prelude::*;
use diesel::prelude::*;
use diesel::r2d2::{self, ConnectionManager};
use dotenv::dotenv;

use crate::models::todo::Todo;
use crate::repository::schema::todos::dsl::*;
```

And then define a type alias for our database connection pool, so we don't have to type it out every time:

```rust
pub type DBPool = r2d2::Pool<ConnectionManager<PgConnection>>;
```

Next, we change our `Database` struct to support the new database connection by adding a `pool` field of type `DBPool`:

```rust
pub struct Database {
    pool: DBPool,
}
```

And then we need to change the `new` function to create a new database connection pool:

```rust
impl Database {
    pub fn new() -> Self {
        dotenv().ok();
        let database_url = std::env::var("DATABASE_URL").expect("DATABASE_URL must be set");
        let manager = ConnectionManager::<PgConnection>::new(database_url);
        let pool: DBPool = r2d2::Pool::builder()
            .build(manager)
            .expect("Failed to create pool.");
        Database { pool }
    }
}
```

We read the database URL from the `.env` file and create a new connection pool with the `r2d2` crate. The `r2d2` is
responsible for managing the database connections and reusing them in a connection pool.

After that, we can change all the functions to use a database.

```rust
impl Database {
    // omitting the rest of the file

    pub fn get_todos(&self) -> Vec<Todo> {
        todos
            .load::<Todo>(&mut self.pool.get().unwrap())
            .expect("Error loading all todos")
    }

    pub fn create_todo(&self, todo: Todo) -> Result<Todo, Error> {
        let todo = Todo {
            id: uuid::Uuid::new_v4().to_string(),
            created_at: Some(Utc::now().naive_utc()),
            updated_at: Some(Utc::now().naive_utc()),
            ..todo
        };
        diesel::insert_into(todos)
            .values(&todo)
            .execute(&mut self.pool.get().unwrap())
            .expect("Error creating new todo");
        Ok(todo)
    }

    pub fn get_todo_by_id(&self, todo_id: &str) -> Option<Todo> {
        let todo = todos
            .find(todo_id)
            .get_result::<Todo>(&mut self.pool.get().unwrap())
            .expect("Error loading todo by id");
        Some(todo)
    }

    pub fn delete_todo_by_id(&self, todo_id: &str) -> Option<usize> {
        let count = diesel::delete(todos.find(todo_id))
            .execute(&mut self.pool.get().unwrap())
            .expect("Error deleting todo by id");
        Some(count)
    }

    pub fn update_todo_by_id(&self, todo_id: &str, mut todo: Todo) -> Option<Todo> {
        todo.updated_at = Some(Utc::now().naive_utc());
        let todo = diesel::update(todos.find(todo_id))
            .set(&todo)
            .get_result::<Todo>(&mut self.pool.get().unwrap())
            .expect("Error updating todo by id");
        Some(todo)
    }
}
```

That was a lot of changes, but it is done, we use now a persistent database for our application.

## Testing the API

Now it is time to test our API with a database backend. Start the application with following command:

```bash
cargo run
```

And execute the following `curl` commands to test the API (marked as Terminal 1). Open a new terminal to run `psql` and
connect to the database to see the changes (marked as Terminal 2).

### Create a new Todo

Terminal 1:

```bash
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy milk", "description": "Buy 2 liters of milk"}' http://localhost:8080/api/todos
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy eggs", "description": "Buy 12 eggs"}' http://localhost:8080/api/todos
curl -X POST -H "Content-Type: application/json" -d '{"title": "Buy bread", "description": "Buy 1 loaf of bread"}' http://localhost:8080/api/todos
```

Terminal 2:

```bash
todo=# SELECT * FROM todos;
                  id                  |   title   |     description      |         created_at         |         updated_at         
--------------------------------------+-----------+----------------------+----------------------------+----------------------------
 087c8867-91d6-4925-b07c-8aa05e811efc | Buy milk  | Buy 2 liters of milk | 2023-03-19 08:54:28.204034 | 2023-03-19 08:54:28.204105
 36bb6fd3-9500-456f-ab90-c9e81acaf108 | Buy eggs  | Buy 12 eggs          | 2023-03-19 09:05:10.713352 | 2023-03-19 09:05:10.713401
 f10baa92-4b0e-4ede-8933-94e2dc0b4843 | Buy bread | Buy 1 loaf of bread  | 2023-03-19 09:05:10.72499  | 2023-03-19 09:05:10.724992
(3 rows)
```

### Get all Todos

Terminal 1:

```bash
curl -s http://localhost:8080/api/todos | jq
[
  {
    "id": "087c8867-91d6-4925-b07c-8aa05e811efc",
    "title": "Buy milk",
    "description": "Buy 2 liters of milk",
    "created_at": "2023-03-19T08:54:28.204034",
    "updated_at": "2023-03-19T08:54:28.204105"
  },
  {
    "id": "36bb6fd3-9500-456f-ab90-c9e81acaf108",
    "title": "Buy eggs",
    "description": "Buy 12 eggs",
    "created_at": "2023-03-19T09:05:10.713352",
    "updated_at": "2023-03-19T09:05:10.713401"
  },
  {
    "id": "f10baa92-4b0e-4ede-8933-94e2dc0b4843",
    "title": "Buy bread",
    "description": "Buy 1 loaf of bread",
    "created_at": "2023-03-19T09:05:10.724990",
    "updated_at": "2023-03-19T09:05:10.724992"
  }
]
```

### Update a Todo

Terminal 1:

```bash
curl -s -X PUT -H "Content-Type: application/json" -d '{"title": "Buy milk", "description": "Buy 20 liters of milk"}' http://localhost:8080/api/todos/087c8867-91d6-4925-b07c-8aa05e811efc | jq
{
  "id": "087c8867-91d6-4925-b07c-8aa05e811efc",
  "title": "Buy milk",
  "description": "Buy 20 liters of milk",
  "created_at": "2023-03-19T08:54:28.204034",
  "updated_at": "2023-03-19T09:07:09.996788"
}
```

Terminal 2:

```bash
todo=#  SELECT * FROM todos WHERE id='087c8867-91d6-4925-b07c-8aa05e811efc';
                  id                  |  title   |      description      |         created_at         |         updated_at         
--------------------------------------+----------+-----------------------+----------------------------+----------------------------
 087c8867-91d6-4925-b07c-8aa05e811efc | Buy milk | Buy 20 liters of milk | 2023-03-19 08:54:28.204034 | 2023-03-19 09:07:09.996788
(1 row)
```

### Delete a Todo

Terminal 1:

```bash
curl -s -X DELETE http://localhost:8080/api/todos/087c8867-91d6-4925-b07c-8aa05e811efc

# Get all todos

curl -s http://localhost:8080/api/todos | jq
[
  {
    "id": "36bb6fd3-9500-456f-ab90-c9e81acaf108",
    "title": "Buy eggs",
    "description": "Buy 12 eggs",
    "created_at": "2023-03-19T09:05:10.713352",
    "updated_at": "2023-03-19T09:05:10.713401"
  },
  {
    "id": "f10baa92-4b0e-4ede-8933-94e2dc0b4843",
    "title": "Buy bread",
    "description": "Buy 1 loaf of bread",
    "created_at": "2023-03-19T09:05:10.724990",
    "updated_at": "2023-03-19T09:05:10.724992"
  }
]
```

Terminal 2:

```bash
todo=# SELECT * FROM todos;
                  id                  |   title   |     description     |         created_at         |         updated_at         
--------------------------------------+-----------+---------------------+----------------------------+----------------------------
 36bb6fd3-9500-456f-ab90-c9e81acaf108 | Buy eggs  | Buy 12 eggs         | 2023-03-19 09:05:10.713352 | 2023-03-19 09:05:10.713401
 f10baa92-4b0e-4ede-8933-94e2dc0b4843 | Buy bread | Buy 1 loaf of bread | 2023-03-19 09:05:10.72499  | 2023-03-19 09:05:10.724992
(2 rows)
```

## Housekeeping

To stop and remove the postgres container and volume, run the following commands:

```bash
docker-compose -f postgres.yaml down
docker volume rm rust-actix-web-rest-api-diesel_postgres-data
```

## Conclusion

Congratulations! We have successfully enhanced your Todo API with a `PostgreSQL` database. You have also learned how to
use the `diesel` crate to interact with the database.

What should I do next? Add authentication and authorization to our API, add a frontend to your API or add OpenTelemetry?
Leave a comment below. I would love to hear from you.

## Resources

* [Diesel](https://diesel.rs/)
* Docker Hub: [postgres](https://hub.docker.com/_/postgres)
* [Docker Compose](https://docs.docker.com/compose/)
* https://actix.rs/
