// @generated automatically by Diesel CLI.

diesel::table! {
    categories (id) {
        id -> Int4,
        name -> Varchar,
        description -> Nullable<Text>,
    }
}

diesel::table! {
    todos (id) {
        id -> Varchar,
        title -> Varchar,
        description -> Nullable<Text>,
        created_at -> Nullable<Timestamp>,
        updated_at -> Nullable<Timestamp>,
        category_id -> Nullable<Int4>,
    }
}

diesel::joinable!(todos -> categories (category_id));

diesel::allow_tables_to_appear_in_same_query!(
    categories,
    todos,
);
