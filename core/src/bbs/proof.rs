use ark_bls12_381::Bls12_381;
use bbs_plus::proof::PoKOfSignatureG1Proof;

pub type BbsProof = PoKOfSignatureG1Proof<Bls12_381>;
