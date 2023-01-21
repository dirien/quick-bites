use reqwest::{Client, Error};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Product {
    pub id: i64,
    pub title: String,
    pub description: String,
    pub price: i64,
    pub rating: f64,
    pub stock: i64,
    pub brand: String,
    pub category: String,
    pub thumbnail: String,
    pub images: Vec<String>,
}


#[tokio::main]
async fn main() -> Result<(), Error> {
    let product: Product = Client::new()
        .get("https://dummyjson.com/products/1")
        .send()
        .await?
        .json()
        .await?;
    println!("{:#?}", product);


    let new_product = Product {
        id: 1,
        title: "Macbook Pro".to_owned(),
        description: "Best laptop ever".to_owned(),
        price: 100,
        rating: 0.0,
        stock: 100,
        brand: "Apple".to_owned(),
        category: "laptops".to_owned(),
        thumbnail: "https://dummyimage.com/300x300/000/fff".to_owned(),
        images: vec![
            "https://dummyimage.com/300x300/000/fff".to_owned(),
            "https://dummyimage.com/600x600/000/fff".to_owned(),
        ],
    };

    let new_product: Product = Client::new()
        .post("https://dummyjson.com/products/add")
        .header("Content-Type", "application/json")
        .json(&new_product)
        .send()
        .await?
        .json()
        .await?;
    println!("{:#?}", new_product);


    let new_product_arbitary_json: Value = Client::new()
        .post("https://dummyjson.com/products/add")
        .header("Content-Type", "application/json")
        .json(&json!({
            "id": 1,
            "title": "Macbook Pro",
            "description": "Best laptop ever",
            "price": 100,
            "rating": 0.0,
            "stock": 100,
            "brand": "Apple",
            "category": "laptops",
            "thumbnail": "https://dummyimage.com/300x300/000/fff",
            "images": [
                "https://dummyimage.com/300x300/000/fff",
                "https://dummyimage.com/600x600/000/fff"
            ]
        }))
        .send()
        .await?
        .json()
        .await?;

    println!("{:#?}", new_product_arbitary_json);

    let cloned_product = new_product_arbitary_json.clone();

    let test: Product = serde_json::from_value(new_product_arbitary_json).unwrap();
    println!("{:#?}", test);

    let description = cloned_product.get("description");
    match description {
        Some(d) => println!("Description: {}", d),
        None => println!("No description"),
    }
    Ok(())
}
