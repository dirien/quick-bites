use serde::{Deserialize, Serialize};
use diesel::{Queryable, Insertable, AsChangeset, Associations, Identifiable};

#[derive(Serialize, Deserialize, Debug, Clone, Queryable, Insertable, AsChangeset, Identifiable)]
#[diesel(table_name = crate::repository::schema::categories)]
pub struct Category {
    pub id: i32,
    pub name: String,
    pub description: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone, Queryable, Insertable, AsChangeset, Associations, Identifiable)]
#[diesel(table_name = crate::repository::schema::todos)]
#[diesel(belongs_to(Category))]
pub struct Todo {
    #[serde(default)]
    pub id: String,
    pub title: String,
    pub description: Option<String>,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
    pub category_id: Option<i32>,
}

#[derive(Serialize)]
pub struct CategoryData {
    pub id: i32,
    pub name: String,
    pub description: Option<String>,
}

#[derive(Serialize)]
pub struct TodoItemData {
    pub id: String,
    pub title: String,
    pub description: Option<String>,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
    pub category: Option<CategoryData>,
}
