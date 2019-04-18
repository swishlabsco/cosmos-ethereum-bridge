pragma solidity ^0.5.0;

import "./CosmosERC20.sol";
import "./Valset.sol";

contract Peggy is Valset {

    /* Global variables  */

    address public relayer;
    uint256 public nonce;
    mapping(bytes32 => bool) public hashes;
    mapping(address => bool) public cosmosTokenAddresses;
    mapping(string => address) cosmosTokens;

    /* Events  */

    event NewCosmosERC20(string name, address tokenAddress);
    event Lock(bytes to, address token, uint64 value, uint256 nonce);
    event Unlock(address to, address token, uint64 value, uint256 nonce);

    /* Modifiers  */

    /*
    * @dev: Modifier to restrict access to the operator.
    *
    */
    modifier onlyRelayer()
    {
        require(
            msg.sender == relayer,
            'Must be the specified relayer.'
        );
        _;
    }

    /*
    * @dev: Constructor, initalizes relayer, initial addresses, and initial powers.
    *
    * @param initAddress: Initial addresses to serve as validators.
    * @param initPowers: Initial validator powers.
    */
    constructor(
        address[] memory initAddress,
        uint64[] memory initPowers
    )
        public
        Valset(initAddress, initPowers)
    {
        relayer = msg.sender;
    }

    /* 
     * @dev: Locks received funds to the consensus of the peg zone.
     *
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
        //Confirm that nonce is available
        require(nonce + 1 > nonce);

        //Actions based on token address type
        if (msg.value != 0) {
          require(tokenAddr == address(0));
          require(msg.value == amount);
        } else if (cosmosTokenAddresses[tokenAddr]) {
          CosmosERC20(tokenAddr).burn(msg.sender, amount);
        } else {
          require(ERC20(tokenAddr).transferFrom(msg.sender, address(this), amount));
        }

        //Increment global nonce
        nonce = nonce + 1;

        //Emit lock event
        emit Lock(to, tokenAddr, amount, nonce);
        return true;
    }


    /*
     * @dev: Unlocks Ethereum tokens according to the information from the pegzone. Called by the relayers.
     *
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
        address payable _to,
        address _token,
        uint64 _amount,
        uint256 _nonce,
        uint[] calldata _signers,
        uint8[] calldata _v,
        bytes32[] calldata _r,
        bytes32[] calldata _s
    )
        onlyRelayer
        external
        returns (bool)
    {
        //Validate that this hash hasn't been used yet
        bytes32 signedHash = processHash(
            _to,
            _token,
            _amount,
            _nonce
        );

        //Check that the hash has enough validated signing power
        require(Valset.verifyValidators(
            signedHash,
            _signers,
            _v,
            _r,
            _s)
        );

        //Actions based on token address type
        if (_token == address(0)) {
          _to.transfer(_amount);
        } else if (cosmosTokenAddresses[_token]) {
          CosmosERC20(_token).mint(_to, _amount);
        } else {
          require(ERC20(_token).transfer(_to, _amount));
        }

        //Emit unlock event
        emit Unlock(_to, _token, _amount, _nonce);
        return true;
    }

    /*
    * @dev: Recreates the original hash from the supplied parameters,
    *       validates that it is unique, then updates the loan registry.
    *
    * @param _to: address of the intended recipient on Ethereum.
    * @param _tokenAddr: token address of the currency on Ethereum (0x0 for eth).
    * @param _amount: value of the transaction.
    * @param _nonce: the transaction's relay nonce.
    * @return: The recreated hash as a bytes32.
    */
    function processHash(
        address _to,
        address _tokenAddr,
        uint64 _amount,
        uint256 _nonce
    )
        internal
        returns (bytes32)
    {
        bytes32 hash = recreateHash(
            _to,
            _tokenAddr,
            _amount,
            _nonce
        );

        if(!hashes[hash]) {
            hashes[hash] = true;
            return hash;
        } else {
            revert("This hash has already been used.");
        }
    }

    /*
    * @dev: Recreates a hash from two integer values.
    *
    * @param _recipient: address of the intended recipient on Cosmos.
    * @param _sender: address of the original sender on Ethereum.
    * @param _amount: value of the transaction.
    * @param _nonce: the transaction's relay nonce.
    * @return: The recreated hash as a bytes32.
    */
    function recreateHash(
        address _recipient,
        address _sender,
        uint64 _amount,
        uint256 _nonce
    )
        internal
        pure
        returns (bytes32)
    {
        return keccak256(
            abi.encodePacked(
                _recipient,
                _sender,
                _amount,
                _nonce
            )
        );
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
        cosmosTokens[name] = address(newToken);
        cosmosTokenAddresses[address(newToken)] = true;

        emit NewCosmosERC20(name, address(newToken));
        return address(newToken);
    }

    /* Helper functions */

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
        address _to,
        address _token,
        uint64 _amount,
        uint256 _nonce
    )
        public
        pure
        returns (bytes32 hash)
    {
        return keccak256(abi.encodePacked(_to, _token, _amount, _nonce));
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
}
