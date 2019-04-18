pragma solidity ^0.5.0;

import "./ECDSA.sol";

contract Valset {

    using ECDSA for bytes32;

    /* Variables */

    address[] public addresses;
    uint64[] public powers;
    uint64 public totalPower;
    uint internal updateSeq = 0;


    /* Events */

    event Update(address[] newAddresses, uint64[] newPowers, uint indexed seq);
    
    /*
    * @dev: Constructor, initalizes relayer, initial addresses, and initial powers.
    *
    * @param initAddress: Initial addresses to serve as validators.
    * @param initPowers: Initial validator powers.
    */    constructor(
        address[] memory initAddress,
        uint64[] memory initPowers
    )
        public
    {
        updateInternal(initAddress, initPowers);
    }

    /* Functions */

    function hashValidatorArrays(
        address[] memory addressesArr,
        uint64[] memory powersArr
    )
        public
        pure
        returns (bytes32 hash)
    {
        return keccak256(abi.encodePacked(addressesArr, powersArr));
    }

    //Confirm that each validator has signed (in order)
    function verifyValidators(
        bytes32 hash,
        uint[] memory signers,
        uint8[] memory v,
        bytes32[] memory r,
        bytes32[] memory s
    )
        public
        view
        returns (bool)
    {
        uint64 signedPower = 0;

        for (uint i = 0; i < signers.length; i++) {
          if (i > 0) {
            require(signers[i] > signers[i-1]);
          }
          //Signatory address must match the specified validator
          if(addresses[signers[i]] == safeRecover(hash, v[i], r[i], s[i])) {
            //Add this validator's signing power to the total
            signedPower += powers[signers[i]];
          } //return hash.toEthSignedMessageHash(); //return hash.recover(signature);

        }
        //Combined signing power must be at least 66.6% of total power
        require(signedPower * 3 > totalPower * 2);
        return true;
    }

    /*
    * @dev: Safely applies ECDSA to signature recovery
    *
    * @param hash: The hashed message which has been signed.
    * @param v: Component of the decomposed signature.
    * @param r: Component of the decomposed signature.
    * @param s: Component of the decomposed signature.
    */
    function safeRecover(
        bytes32 hash,
        uint8 v,
        bytes32 r,
        bytes32 s
    )
        internal
        pure
        returns(address)
    {
        if (uint256(s) > 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0) {
            return address(0);
        }

        if (v != 27 && v != 28) {
            return address(0);
        }

        // If the signature is valid (and not malleable), return the signer address
        return ecrecover(hash, v, r, s);         
    }

    //Set new list of validators and their respective signing power
    function updateInternal(
        address[] memory newAddress,
        uint64[] memory newPowers
    )
        internal
        returns (bool)
    {   
        //Initalize empty arrays for validators and signing powers
        addresses = new address[](newAddress.length);
        powers    = new uint64[](newPowers.length);

        //Reset total power
        totalPower = 0;
        //Set each address and power, increment total power
        for (uint i = 0; i < newAddress.length; i++) {
            addresses[i] = newAddress[i];
            powers[i]    = newPowers[i];
            totalPower  += newPowers[i];
        }
        uint updateCount = updateSeq;
        emit Update(addresses, powers, updateCount);
        updateSeq++;
        return true;
    }


    /// Updates validator set. Called by the relayers.
    /*
     * @param newAddress  new validators addresses
     * @param newPower    power of each validator
     * @param signers     indexes of each signer validator
     * @param v           recovery id. Used to compute ecrecover
     * @param r           output of ECDSA signature. Used to compute ecrecover
     * @param s           output of ECDSA signature.  Used to compute ecrecover
     */
    function update(
        address[] memory newAddress,
        uint64[] memory newPowers,
        uint[] memory signers,
        uint8[] memory v,
        bytes32[] memory r,
        bytes32[] memory s
    )
        public
    {
        bytes32 hashData = keccak256(abi.encodePacked(newAddress, newPowers));
        require(verifyValidators(hashData, signers, v, r, s));
        require(updateInternal(newAddress, newPowers));
    }

    /* Getters */

    function getAddresses()
        public
        view
        returns (address[] memory)
    {
        return addresses;
    }

    function getPowers()
        public
        view
        returns (uint64[] memory)
    {
        return powers;
    }

    function getTotalPower()
        public
        view
        returns (uint64)
    {
        return totalPower;
    }
}
