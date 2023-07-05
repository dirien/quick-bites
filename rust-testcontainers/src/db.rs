use std::str::FromStr;
use mongodb::bson::{doc, oid::ObjectId};
use mongodb::{Client, Collection, options::ClientOptions};
use crate::config::Config;

use crate::model::Car;

const DB_NAME: &str = "cars_info";
const COLLECTION_NAME: &str = "cars";

#[derive(Clone, Debug)]
pub struct MongoDbClient {
    client: Client,
}

impl MongoDbClient {
    pub async fn new(config: Config) -> Self {
        let client_options = ClientOptions::parse(&config.mongodb_uri).await.unwrap();
        let mongodb_client = Client::with_options(client_options).expect("Failed to create MongoDB client");
        Self {
            client: mongodb_client,
        }
    }
    pub async fn get_car(&self, car_id: &str) -> Result<Car, anyhow::Error> {
        let id = ObjectId::from_str(car_id)
            .map_err(|e| anyhow::anyhow!("Failed to convert id to ObjectId: {}", e))?;
        let filter = doc! { "_id": id };
        let car = self.get_cars_collection().find_one(filter, None).await?.ok_or(
            anyhow::anyhow!("Car with id {} not found", car_id)
        )?;
        Ok(car)
    }

    pub async fn delete_car(&self, car_id: &str) -> Result<(), anyhow::Error> {
        let id = ObjectId::from_str(car_id)
            .map_err(|e| anyhow::anyhow!("Failed to convert id to ObjectId: {}", e))?;
        let filter = doc! { "_id": id };
        self.get_cars_collection().delete_one(filter, None).await?;
        Ok(())
    }

    pub async fn get_cars(&self) -> Result<Vec<Car>, anyhow::Error> {
        let mut cars = self.get_cars_collection().find(None, None).await?;
        let mut result: Vec<Car> = Vec::new();
        while cars.advance().await? {
            println!("{:?}", cars.current());
            result.push(cars.deserialize_current().unwrap());
        }
        Ok(result)
    }

    pub async fn create_car(&self, car: Car) -> Result<Car, anyhow::Error> {
        let result = self.get_cars_collection().insert_one(car, None).await?;

        let filter = doc! { "_id": result.inserted_id.as_object_id().unwrap().clone() };
        let car = self.get_cars_collection().find_one(filter, None).await?;
        Ok(car.unwrap())
    }

    fn get_cars_collection(&self) -> Collection<Car> {
        let db = self.client.database(DB_NAME);
        db.collection(COLLECTION_NAME)
    }
}
