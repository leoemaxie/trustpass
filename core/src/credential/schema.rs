use std::collections::HashMap;
use ark_bls12_381::Fr;
use chrono::NaiveDate;
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::bbs::signature::attribute_to_scalar;
use crate::error::{CoreError, Result};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum AttributeType {
    #[serde(rename = "string")]
    String,
    #[serde(rename = "date")]
    Date,
    #[serde(rename = "decimal")]
    Decimal,
    #[serde(rename = "integer")]
    Integer,
    #[serde(rename = "boolean")]
    Boolean,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct AttributeDefinition {
    pub name: String,
    #[serde(rename = "type")]
    pub attr_type: AttributeType,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct CredentialSchema {
    pub id: String,
    pub name: String,
    pub version: u32,
    pub attributes: Vec<AttributeDefinition>,
    #[serde(rename = "issuerDid")]
    pub issuer_did: String,
}

impl CredentialSchema {
    /// Returns the attributes sorted canonically by name for deterministic BBS+ message mapping
    pub fn canonical_attributes(&self) -> Vec<AttributeDefinition> {
        let mut sorted = self.attributes.clone();
        sorted.sort_by(|a, b| a.name.cmp(&b.name));
        sorted
    }

    /// Number of attributes in this schema
    pub fn attribute_count(&self) -> usize {
        self.attributes.len()
    }

    /// Validates that a map of claims satisfies this schema's types and required attributes
    pub fn validate_claims(&self, claims: &HashMap<String, Value>) -> Result<()> {
        for attr in &self.attributes {
            let val = claims.get(&attr.name).ok_or_else(|| {
                CoreError::MissingAttribute(format!(
                    "Schema '{}' requires attribute '{}', which was missing",
                    self.name, attr.name
                ))
            })?;

            match attr.attr_type {
                AttributeType::String => {
                    if !val.is_string() {
                        return Err(CoreError::SchemaValidation(format!(
                            "Attribute '{}' expected string, got: {:?}",
                            attr.name, val
                        )));
                    }
                }
                AttributeType::Date => {
                    let s = val.as_str().ok_or_else(|| {
                        CoreError::SchemaValidation(format!(
                            "Attribute '{}' expected date string, got: {:?}",
                            attr.name, val
                        ))
                    })?;
                    NaiveDate::parse_from_str(s, "%Y-%m-%d").map_err(|e| {
                        CoreError::SchemaValidation(format!(
                            "Attribute '{}' date must be YYYY-MM-DD: {}",
                            attr.name, e
                        ))
                    })?;
                }
                AttributeType::Decimal => {
                    if !val.is_number() {
                        return Err(CoreError::SchemaValidation(format!(
                            "Attribute '{}' expected decimal number, got: {:?}",
                            attr.name, val
                        )));
                    }
                }
                AttributeType::Integer => {
                    if !val.is_i64() && !val.is_u64() {
                        return Err(CoreError::SchemaValidation(format!(
                            "Attribute '{}' expected integer number, got: {:?}",
                            attr.name, val
                        )));
                    }
                }
                AttributeType::Boolean => {
                    if !val.is_boolean() {
                        return Err(CoreError::SchemaValidation(format!(
                            "Attribute '{}' expected boolean, got: {:?}",
                            attr.name, val
                        )));
                    }
                }
            }
        }
        Ok(())
    }

    /// Transforms validated claims into canonical Fr message scalars for BBS+ signing
    pub fn claims_to_scalars(&self, claims: &HashMap<String, Value>) -> Result<Vec<Fr>> {
        self.validate_claims(claims)?;
        let canonical = self.canonical_attributes();
        let mut scalars = Vec::with_capacity(canonical.len());

        for attr in canonical {
            let val = claims.get(&attr.name).unwrap();
            let val_str = match val {
                Value::String(s) => s.clone(),
                Value::Number(n) => n.to_string(),
                Value::Bool(b) => b.to_string(),
                other => other.to_string(),
            };
            scalars.push(attribute_to_scalar(&attr.name, &val_str));
        }

        Ok(scalars)
    }

    /// Finds the canonical index of a given attribute name for selective disclosure
    pub fn attribute_index(&self, name: &str) -> Result<usize> {
        let canonical = self.canonical_attributes();
        canonical
            .iter()
            .position(|a| a.name == name)
            .ok_or_else(|| {
                CoreError::MissingAttribute(format!(
                    "Attribute '{}' not found in schema '{}'",
                    name, self.name
                ))
            })
    }
}

/// Seed schema: NationalIDCredential v1
pub fn seed_national_id_schema(issuer_did: &str) -> CredentialSchema {
    CredentialSchema {
        id: "schema:national-id:v1".to_string(),
        name: "NationalIDCredential".to_string(),
        version: 1,
        attributes: vec![
            AttributeDefinition {
                name: "fullName".to_string(),
                attr_type: AttributeType::String,
            },
            AttributeDefinition {
                name: "dateOfBirth".to_string(),
                attr_type: AttributeType::Date,
            },
            AttributeDefinition {
                name: "nationality".to_string(),
                attr_type: AttributeType::String,
            },
            AttributeDefinition {
                name: "idNumber".to_string(),
                attr_type: AttributeType::String,
            },
        ],
        issuer_did: issuer_did.to_string(),
    }
}

/// Seed schema: StudentCredential v1
pub fn seed_student_schema(issuer_did: &str) -> CredentialSchema {
    CredentialSchema {
        id: "schema:student:v1".to_string(),
        name: "StudentCredential".to_string(),
        version: 1,
        attributes: vec![
            AttributeDefinition {
                name: "studentId".to_string(),
                attr_type: AttributeType::String,
            },
            AttributeDefinition {
                name: "university".to_string(),
                attr_type: AttributeType::String,
            },
            AttributeDefinition {
                name: "enrollmentStatus".to_string(),
                attr_type: AttributeType::String,
            },
            AttributeDefinition {
                name: "gpa".to_string(),
                attr_type: AttributeType::Decimal,
            },
        ],
        issuer_did: issuer_did.to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_national_id_schema_validation_success() {
        let schema = seed_national_id_schema("did:key:z123");
        let mut claims = HashMap::new();
        claims.insert("fullName".to_string(), json!("Amina Bello"));
        claims.insert("dateOfBirth".to_string(), json!("1998-04-12"));
        claims.insert("nationality".to_string(), json!("NG"));
        claims.insert("idNumber".to_string(), json!("NIN-83920192"));

        assert!(schema.validate_claims(&claims).is_ok());
        let scalars = schema.claims_to_scalars(&claims).unwrap();
        assert_eq!(scalars.len(), 4);
    }

    #[test]
    fn test_national_id_schema_validation_failures() {
        let schema = seed_national_id_schema("did:key:z123");

        // Missing field
        let mut incomplete = HashMap::new();
        incomplete.insert("fullName".to_string(), json!("Amina Bello"));
        assert!(schema.validate_claims(&incomplete).is_err());

        // Invalid date format
        let mut bad_date = HashMap::new();
        bad_date.insert("fullName".to_string(), json!("Amina Bello"));
        bad_date.insert("dateOfBirth".to_string(), json!("12/04/1998"));
        bad_date.insert("nationality".to_string(), json!("NG"));
        bad_date.insert("idNumber".to_string(), json!("NIN-83920192"));
        assert!(schema.validate_claims(&bad_date).is_err());
    }

    #[test]
    fn test_student_schema_validation() {
        let schema = seed_student_schema("did:key:zUniv");
        let mut claims = HashMap::new();
        claims.insert("studentId".to_string(), json!("STU-2024-001"));
        claims.insert("university".to_string(), json!("University of Lagos"));
        claims.insert("enrollmentStatus".to_string(), json!("active"));
        claims.insert("gpa".to_string(), json!(3.85));

        assert!(schema.validate_claims(&claims).is_ok());

        // Non-decimal GPA
        claims.insert("gpa".to_string(), json!("not_a_number"));
        assert!(schema.validate_claims(&claims).is_err());
    }
}
