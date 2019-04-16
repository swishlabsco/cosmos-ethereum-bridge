pragma solidity ^0.5.0;

contract Valset {

    /* Variables */

    address[] public addresses;
    uint64[] public powers;
    uint64 public totalPower;
    uint internal updateSeq = 0;


    /* Events */

    event Update(address[] newAddresses, uint64[] newPowers, uint indexed seq);


    /* Getters (These are supposed to be auto implemented by solidity but aren't ¯\_(ツ)_/¯) */

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
          address recAddr = ecrecover(hash, v[i], r[i], s[i]);
          require(recAddr == addresses[signers[i]]);

          //Add this validator's signing power to the total
          signedPower += powers[signers[i]];
        }
        //Combined signing power must be at least 66.6% of total power
        require(signedPower * 3 > totalPower * 2);
        return true;
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

    //Sets validators and powers on deployment
    constructor(
        address[] memory initAddress,
        uint64[] memory initPowers
    )
        public
    {
        updateInternal(initAddress, initPowers);
    }
}
