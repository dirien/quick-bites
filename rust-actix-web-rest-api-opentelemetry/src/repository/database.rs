use chrono::prelude::*;
use diesel::prelude::*;
use diesel::r2d2::{self, ConnectionManager};
use std::fmt::Error;

use dotenv::dotenv;

use crate::models::todo::{Category, CategoryData, Todo, TodoItemData};
use crate::repository::schema::categories::dsl::*;
use crate::repository::schema::todos::dsl::*;

type DBPool = r2d2::Pool<ConnectionManager<PgConnection>>;

#[derive(Debug, Clone)]
pub struct Database {
    pool: DBPool,
}

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

    pub fn get_categories(&self) -> Vec<Category> {
        categories
            .load::<Category>(&mut self.pool.get().unwrap())
            .expect("Error loading all categories")
    }

    pub fn get_todos_with_category(&self) -> Vec<TodoItemData> {
        let mut empty_todo_item_data_list: Vec<TodoItemData> = Vec::new();
        todos
            .inner_join(categories)
            .load::<(Todo, Category)>(&mut self.pool.get().unwrap())
            .expect("Error loading all todos")
            .into_iter()
            .for_each(|(todo, category)| {
                println!("todo: {:?}, category: {:?}", todo, category);
                let todo_item_data = TodoItemData {
                    id: todo.id,
                    title: todo.title,
                    description: todo.description,
                    created_at: todo.created_at,
                    updated_at: todo.updated_at,
                    category: Some(CategoryData {
                        id: category.id,
                        name: category.name,
                        description: category.description,
                    }),
                };
                empty_todo_item_data_list.push(todo_item_data);
            });
        empty_todo_item_data_list
    }

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
