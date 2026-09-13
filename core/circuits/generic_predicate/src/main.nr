// TrustPass: Generic Zero-Knowledge Predicate Evaluation Circuit
// Proves that a private credential attribute satisfies a specified operator against a threshold,
// without revealing the private attribute value.
//
// Strictly generic: zero claim-specific or domain-specific functions.

global OP_GTE: u8 = 1;
global OP_EQ: u8 = 2;
global OP_LTE: u8 = 3;
global OP_IN_RANGE: u8 = 4;

fn main(
    private_value: Field,          // Secret attribute (e.g. YYYYMMDD date, GPA fixed-point, salary, hash)
    operator: pub u8,              // OP_GTE (1), OP_EQ (2), OP_LTE (3), OP_IN_RANGE (4)
    threshold: pub Field,          // Primary comparison threshold
    upper_bound: pub Field         // Upper bound (used when operator == OP_IN_RANGE)
) {
    if operator == OP_EQ {
        assert(private_value == threshold);
    } else if operator == OP_GTE {
        // Enforces private_value >= threshold via range constraint on non-negative difference
        let diff = private_value - threshold;
        diff.assert_max_bit_size(64);
    } else if operator == OP_LTE {
        // Enforces private_value <= threshold (used for BEFORE_DATE e.g. DOB <= cutoffDate)
        let diff = threshold - private_value;
        diff.assert_max_bit_size(64);
    } else if operator == OP_IN_RANGE {
        // Enforces threshold <= private_value <= upper_bound
        let diff_low = private_value - threshold;
        diff_low.assert_max_bit_size(64);
        let diff_high = upper_bound - private_value;
        diff_high.assert_max_bit_size(64);
    } else {
        // Unknown operator is rejected
        assert(false);
    }
}

#[test]
fn test_age_check_via_lte() {
    // DOB: 1999-07-20 represented as 19990720
    // 18+ Cutoff: 2008-09-13 represented as 20080913
    // Born on or before cutoff date -> satisfies age >= 18
    main(19990720, OP_LTE, 20080913, 0);
}

#[test]
fn test_gpa_check_via_gte() {
    // GPA 3.82 represented as fixed-point 382
    // Threshold 3.50 represented as 350
    main(382, OP_GTE, 350, 0);
}

#[test]
fn test_citizenship_hash_via_eq() {
    // Hash scalar for "NG"
    let ng_hash = 0x4e47;
    main(ng_hash, OP_EQ, ng_hash, 0);
}

#[test]
fn test_salary_in_range() {
    // Salary 65000 in range [50000, 80000]
    main(65000, OP_IN_RANGE, 50000, 80000);
}

#[test(should_fail)]
fn test_failing_gpa_check() {
    // GPA 3.15 (315) does not satisfy GTE 3.50 (350)
    main(315, OP_GTE, 350, 0);
}
