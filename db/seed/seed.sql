-- TrustPass Seed Data
-- Two seed schemas: NationalIDCredential v1 and StudentCredential v1

INSERT INTO credential_schemas (id, name, version, attributes, issuer_did)
VALUES
(
    '11111111-1111-1111-1111-111111111111',
    'NationalIDCredential',
    1,
    '[
        {"name": "fullName", "type": "string"},
        {"name": "dateOfBirth", "type": "date"},
        {"name": "nationality", "type": "string"},
        {"name": "idNumber", "type": "string"}
    ]'::jsonb,
    'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k'
)
ON CONFLICT (name, version) DO UPDATE SET
    attributes = EXCLUDED.attributes,
    issuer_did = EXCLUDED.issuer_did;

INSERT INTO credential_schemas (id, name, version, attributes, issuer_did)
VALUES
(
    '22222222-2222-2222-2222-222222222222',
    'StudentCredential',
    1,
    '[
        {"name": "studentId", "type": "string"},
        {"name": "university", "type": "string"},
        {"name": "enrollmentStatus", "type": "string"},
        {"name": "gpa", "type": "decimal"}
    ]'::jsonb,
    'did:key:zUC724vsrMwHvKyqDdHtrh7z2GNe5xbsfgivth466P4vm2iaJLW9kK48DbgKa32yL944yK9k'
)
ON CONFLICT (name, version) DO UPDATE SET
    attributes = EXCLUDED.attributes,
    issuer_did = EXCLUDED.issuer_did;
