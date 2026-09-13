use chrono::NaiveDate;
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::error::{CoreError, Result};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum PredicateOp {
    #[serde(rename = "GTE")]
    GTE,
    #[serde(rename = "EQ")]
    EQ,
    #[serde(rename = "IN_SET")]
    InSet,
    #[serde(rename = "BEFORE_DATE")]
    BeforeDate,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClaimRequest {
    #[serde(rename = "schemaName")]
    pub schema_name: String,
    #[serde(rename = "attributeName")]
    pub attribute_name: String,
    pub operator: PredicateOp,
    pub value: Value,
}

/// Generic predicate evaluation engine: evaluatePredicate(attribute, operator, threshold)
/// Strictly claim-agnostic — operates over schema attributes and operators.
pub fn evaluate_predicate(
    attribute: &Value,
    operator: &PredicateOp,
    threshold: &Value,
) -> Result<bool> {
    match operator {
        PredicateOp::EQ => Ok(attribute == threshold),

        PredicateOp::GTE => {
            // Compare as numbers (supports JSON numbers and numeric strings generically)
            let attr_num = attribute
                .as_f64()
                .or_else(|| attribute.as_str().and_then(|s| s.parse::<f64>().ok()));
            let thresh_num = threshold
                .as_f64()
                .or_else(|| threshold.as_str().and_then(|s| s.parse::<f64>().ok()));

            match (attr_num, thresh_num) {
                (Some(attr_val), Some(thresh_val)) => Ok(attr_val >= thresh_val),
                _ => Err(CoreError::InvalidPredicate(format!(
                    "GTE requires numeric values, got attribute: {:?}, threshold: {:?}",
                    attribute, threshold
                ))),
            }
        }

        PredicateOp::InSet => {
            let set = threshold.as_array().ok_or_else(|| {
                CoreError::InvalidPredicate(format!(
                    "IN_SET requires an array threshold, got: {:?}",
                    threshold
                ))
            })?;
            Ok(set.contains(attribute))
        }

        PredicateOp::BeforeDate => {
            let attr_str = attribute.as_str().ok_or_else(|| {
                CoreError::InvalidPredicate(format!(
                    "BEFORE_DATE requires string date attribute, got: {:?}",
                    attribute
                ))
            })?;
            let thresh_str = threshold.as_str().ok_or_else(|| {
                CoreError::InvalidPredicate(format!(
                    "BEFORE_DATE requires string date threshold, got: {:?}",
                    threshold
                ))
            })?;

            let attr_date = NaiveDate::parse_from_str(attr_str, "%Y-%m-%d")
                .map_err(|e| CoreError::InvalidPredicate(format!("Failed to parse attribute date '{}': {}", attr_str, e)))?;
            let thresh_date = NaiveDate::parse_from_str(thresh_str, "%Y-%m-%d")
                .map_err(|e| CoreError::InvalidPredicate(format!("Failed to parse threshold date '{}': {}", thresh_str, e)))?;

            // attribute <= threshold means the date occurred on or before the threshold date
            // (e.g. dateOfBirth <= 2008-09-13 implies age >= 18)
            Ok(attr_date <= thresh_date)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_evaluate_predicate_eq() {
        let attr = json!("NG");
        assert!(evaluate_predicate(&attr, &PredicateOp::EQ, &json!("NG")).unwrap());
        assert!(!evaluate_predicate(&attr, &PredicateOp::EQ, &json!("US")).unwrap());
    }

    #[test]
    fn test_evaluate_predicate_gte() {
        let gpa = json!(3.75);
        assert!(evaluate_predicate(&gpa, &PredicateOp::GTE, &json!(3.50)).unwrap());
        assert!(evaluate_predicate(&gpa, &PredicateOp::GTE, &json!(3.75)).unwrap());
        assert!(!evaluate_predicate(&gpa, &PredicateOp::GTE, &json!(3.80)).unwrap());
    }

    #[test]
    fn test_evaluate_predicate_in_set() {
        let status = json!("active");
        let valid_statuses = json!(["active", "enrolled", "probationary"]);
        assert!(evaluate_predicate(&status, &PredicateOp::InSet, &valid_statuses).unwrap());

        let invalid_status = json!("graduated");
        assert!(!evaluate_predicate(&invalid_status, &PredicateOp::InSet, &valid_statuses).unwrap());
    }

    #[test]
    fn test_evaluate_predicate_before_date() {
        // DOB: 2000-01-01 is before 2008-09-13 (satisfies age >= 18)
        let dob_adult = json!("2000-01-01");
        let cutoff = json!("2008-09-13");
        assert!(evaluate_predicate(&dob_adult, &PredicateOp::BeforeDate, &cutoff).unwrap());

        // DOB: 2010-05-20 is after 2008-09-13 (under 18, does not satisfy)
        let dob_minor = json!("2010-05-20");
        assert!(!evaluate_predicate(&dob_minor, &PredicateOp::BeforeDate, &cutoff).unwrap());
    }
}
