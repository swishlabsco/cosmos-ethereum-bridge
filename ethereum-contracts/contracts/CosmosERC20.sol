pragma solidity ^0.5.0;

import "./ERC20.sol";
import "./SafeMath.sol";

contract CosmosERC20 is ERC20 {

    using SafeMath for uint;

    string public name;
    uint public decimals;
    uint private _totalSupply;

    mapping (address => uint) balances;
    mapping (address => mapping (address => uint)) allowed;

    address public controller;

    event Mint(address _to, uint _amount);
    event Burn(address _from, uint _amount);


    modifier onlyByController() {
        require(msg.sender == controller);
        _;
    }

    //Must specify 'memory' because type string = bytes[]
    function symbol()
        public
        view
        returns (string memory)
    {
        return name;
    }

    function totalSupply()
        public
        view
        returns (uint)
    {
        return _totalSupply;
    }

    function balanceOf(
        address tokenOwner
    )
        public
        view
        returns (uint balance)
    {
        return balances[tokenOwner];
    }

    function allowance(
        address tokenOwner,
        address spender
    )
        public
        view
        returns (uint remaining)
    {
        return allowed[tokenOwner][spender];
    }

    function transfer(
        address to,
        uint amount
    )
        public
        returns (bool success)
    {
        return transferFrom(msg.sender, to, amount);
    }

    function approve(
        address spender,
        uint amount
    )
        public
        returns (bool success)
    {
        allowed[msg.sender][spender] = amount;
        emit Approval(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(
        address from,
        address to,
        uint amount
    )
        public
        returns (bool success)
    {
        require(to != controller);
        require(to != address(this));
        balances[from] = balances[from].sub(amount);
        if (from != msg.sender) {
          allowed[from][msg.sender] = allowed[from][msg.sender].sub(amount);
        }
        balances[to] = balances[to].add(amount);

        emit Transfer(from, to, amount);
        return true;
    }

    function mint(
        address to,
        uint amount
    )
        public
        onlyByController()
        returns (bool success)
    {
        balances[to] = balances[to].add(amount);
        _totalSupply = _totalSupply.add(amount);
        emit Mint(to, amount);
        return true;
    }

    function burn(
        address from,
        uint amount
    )
        public
        onlyByController()
        returns (bool success)
    {
        require(balances[from] >= amount);
        balances[from] = balances[from].sub(amount);
        _totalSupply = _totalSupply.sub(amount);
        emit Burn(from, amount);
        return true;
    }

    constructor(
        address _controller,
        string memory _name,
        uint _decimals
    )
        public
    {
        _totalSupply = 0;
        controller = _controller;
        name = _name;
        decimals = _decimals;
    }
}
