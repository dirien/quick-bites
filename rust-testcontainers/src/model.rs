use mongodb::bson::{self, doc, oid::ObjectId, Document};
use serde::{Deserialize, Serialize};
use std::fmt;
use std::fmt::Formatter;
use std::str::FromStr;
use crate::dto::CarDto;

#[derive(Debug, Serialize, Deserialize)]
pub struct Car {
    #[serde(rename = "_id", skip_serializing_if = "Option::is_none")]
    pub id: Option<ObjectId>,
    pub name: String,
    pub brand: String,
    pub year: i32,
    pub r#type: CarType,
}

#[derive(Copy, Clone, Eq, PartialEq, Serialize, Deserialize, Debug)]
pub enum CarType {
    Sedan,
    Hatchback,
    SUV,
    Crossover,
    Coupe,
    Convertible,
    Minivan,
    Pickup,
    Van,
    Wagon,
    Other,
}

impl From<&Car> for Document {
    fn from(value: &Car) -> Self {
        bson::to_document(value).expect("Failed to convert Car to Document")
    }
}

impl From<CarDto> for Car {
    fn from(value: CarDto) -> Self {
        Self {
            id: value.id.map(|id| ObjectId::from_str(id.as_str()).expect("Failed to convert id to ObjectId")),
            name: value.name,
            brand: value.brand,
            year: value.year,
            r#type: value.r#type,
        }
    }
}

impl fmt::Display for CarType {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}
