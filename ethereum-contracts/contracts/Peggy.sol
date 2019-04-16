pragma solidity ^0.5.0;

import "./CosmosERC20.sol";
import "./Valset.sol";

contract Peggy is Valset {

    mapping (string => address) cosmosTokens;
    mapping (address => bool) cosmosTokenAddresses;

    /* Events  */
    event NewCosmosERC20(string name, address tokenAddress);
    event Lock(bytes to, address token, uint64 value);
    event Unlock(address to, address token, uint64 value);

    /* Functions */

    function hashNewCosmosERC20(
        string memory name,
        uint decimals
    )
        public
        pure
        returns (bytes32 hash)
    {
        return keccak256(abi.encodePacked(name, decimals));
    }

    function hashUnlock(
        address to,
        address token,
        uint64 amount
    )
        public
        pure
        returns (bytes32 hash)
    {
        return keccak256(abi.encodePacked(to, token, amount));
    }

    function getCosmosTokenAddress(
        string memory name
    )
        public
        view
        returns (address addr)
    {
        return cosmosTokens[name];
    }


    function isCosmosTokenAddress(
        address addr
    )
        public
        view
        returns (bool isCosmosAddr)
    {
        return cosmosTokenAddresses[addr];
    }

    // Locks received funds to the consensus of the peg zone
    /*
     * @param to          bytes representation of destination address
     * @param value       value of transference
     * @param token       token address in origin chain (0x0 if Ethereum, Cosmos for other values)
     */
    function lock(
        bytes memory to,
        address tokenAddr,
        uint64 amount
    )
        public
        payable
        returns (bool)
    {
        if (msg.value != 0) {
          require(tokenAddr == address(0));
          require(msg.value == amount);
        } else if (cosmosTokenAddresses[tokenAddr]) {
          CosmosERC20(tokenAddr).burn(msg.sender, amount);
        } else {
          require(ERC20(tokenAddr).transferFrom(msg.sender, address(this), amount));
        }
        emit Lock(to, tokenAddr, amount);
        return true;
    }

    // Unlocks Ethereum tokens according to the information from the pegzone. Called by the relayers.
    /*
     * @param to          bytes representation of destination address
     * @param value       value of transference
     * @param token       token address in origin chain (0x0 if Ethereum, Cosmos for other values)
     * @param chain       bytes respresentation of the destination chain (not used in MVP, for incentivization of relayers)
     * @param signers     indexes of each validator
     * @param v           array of recoverys id
     * @param r           array of outputs of ECDSA signature
     * @param s           array of outputs of ECDSA signature
     */
    function unlock(
        address payable to,
        address token,
        uint64 amount,
        uint[] calldata signers,
        uint8[] calldata v,
        bytes32[] calldata r,
        bytes32[] calldata s
    )
        external
        returns (bool)
    {
        bytes32 hashData = keccak256(abi.encodePacked(to, token, amount));
        require(Valset.verifyValidators(hashData, signers, v, r, s));
        if (token == address(0)) {
          to.transfer(amount);
        } else if (cosmosTokenAddresses[token]) {
          CosmosERC20(token).mint(to, amount);
        } else {
          require(ERC20(token).transfer(to, amount));
        }
        emit Unlock(to, token, amount);
        return true;
    }
    //Enables validators to add a new CosmosERC20 to the mapping, enabling support
    function newCosmosERC20(
        string calldata name,
        uint decimals,
        uint[] calldata signers,
        uint8[] calldata v,
        bytes32[] calldata r,
        bytes32[] calldata s
    )
        external
        returns (address addr)
    {
        require(cosmosTokens[name] == address(0));

        bytes32 hashData = keccak256(abi.encodePacked(name, decimals));
        require(Valset.verifyValidators(hashData, signers, v, r, s));

        CosmosERC20 newToken = new CosmosERC20(address(this), name, decimals);

        //Note: address(newToken) casting required for v0.5.0.
        //
        //      address newTokenAddr = address(newToken);
        //      ^ Error: "Stack too deep, try removing local variables".

        cosmosTokens[name] = address(newToken);
        cosmosTokenAddresses[address(newToken)] = true;

        emit NewCosmosERC20(name, address(newToken));
        return address(newToken);
    }

    constructor(
        address[] memory initAddress,
        uint64[] memory initPowers
    )
        public
        Valset(initAddress, initPowers)
    {
        //Intentionally blank
    }
}
