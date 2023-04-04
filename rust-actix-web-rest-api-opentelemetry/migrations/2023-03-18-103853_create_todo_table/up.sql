CREATE SEQUENCE categories_id_seq;

CREATE TABLE categories
(
    id          INTEGER PRIMARY KEY DEFAULT nextval('categories_id_seq'),
    name        VARCHAR(255) NOT NULL,
    description TEXT
);

CREATE TABLE todos
(
    id          VARCHAR(255) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMP,
    updated_at  TIMESTAMP,
    category_id INTEGER,
    FOREIGN KEY (category_id) REFERENCES categories (id)
);

INSERT INTO categories (name, description)
VALUES ('Work', 'Tasks related to work or job responsibilities'),
       ('Personal', 'Personal tasks and errands'),
       ('Health', 'Health and fitness related tasks'),
       ('Hobbies', 'Tasks related to hobbies and interests'),
       ('Education', 'Tasks related to learning and education');
