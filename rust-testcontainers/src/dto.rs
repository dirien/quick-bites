use crate::model::{Car, CarType};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct CarDto {
    pub id: Option<String>,
    pub name: String,
    pub brand: String,
    pub year: i32,
    pub r#type: CarType,
}

#[derive(Serialize)]
pub struct CarMessage {
    pub id: Option<String>,
    pub name: String,
    pub r#type: CarType,
}


impl From<&Car> for CarDto {
    fn from(value: &Car) -> Self {
        Self {
            id: value.id.map(|id| id.to_string()),
            name: value.name.clone(),
            brand: value.brand.clone(),
            year: value.year,
            r#type: value.r#type,
        }
    }
}

impl From<&Car> for CarMessage {
    fn from(value: &Car) -> Self {
        Self {
            id: value.id.map(|id| id.to_string()),
            name: value.name.clone(),
            r#type: value.r#type,
        }
    }
}
