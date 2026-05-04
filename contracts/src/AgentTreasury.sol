// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract AgentTreasury is ERC20, Ownable, ReentrancyGuard {
    error SlippageExceeded();
    error InsufficientPayment();
    error InsufficientOperationalBalance();
    error RecipientNotWhitelisted();
    error Unauthorized();

    event SharesBought(address indexed buyer, uint256 amount, uint256 paid);
    event SharesSold(address indexed seller, uint256 amount, uint256 received);
    event RevenueDistributed(uint256 amount, uint256 operationalBps, uint256 investorBps, uint256 platformBps);
    event DividendsClaimed(address indexed investor, uint256 amount);
    event OpsSpend(address indexed recipient, uint256 amount, string reason);
    event RecipientWhitelistUpdated(address indexed recipient, bool allowed);
    event OrchestratorUpdated(address indexed orchestrator);

    uint256 public constant BPS_DENOMINATOR = 10_000;
    uint256 public constant OPERATIONAL_BPS = 7_000;
    uint256 public constant INVESTOR_BPS = 2_000;
    uint256 public constant PLATFORM_BPS = 1_000;
    uint256 private constant ACC_PRECISION = 1e36;

    uint256 public immutable tokenId;
    uint256 public immutable basePrice;
    uint256 public immutable slope;
    address public immutable platformWallet;

    address public orchestrator;
    uint256 public operationalBalance;
    uint256 public investorPool;
    uint256 public accDividendPerShare;

    mapping(address => bool) public whitelistedRecipients;
    mapping(address => uint256) public dividendDebt;

    modifier onlyAgentOwnerOrOrchestrator() {
        if (msg.sender != owner() && msg.sender != orchestrator) revert Unauthorized();
        _;
    }

    constructor(
        uint256 tokenId_,
        address agentOwner,
        address platformWallet_,
        address orchestrator_,
        uint256 basePrice_,
        uint256 slope_
    ) ERC20("AetherNet Agent Share", "AAS") Ownable(agentOwner) {
        tokenId = tokenId_;
        platformWallet = platformWallet_;
        orchestrator = orchestrator_;
        basePrice = basePrice_;
        slope = slope_;
    }

    receive() external payable {
        operationalBalance += msg.value;
    }

    function decimals() public pure override returns (uint8) {
        return 0;
    }

    function setOrchestrator(address nextOrchestrator) external onlyOwner {
        orchestrator = nextOrchestrator;
        emit OrchestratorUpdated(nextOrchestrator);
    }

    function setWhitelistedRecipient(address recipient, bool allowed) external onlyOwner {
        whitelistedRecipients[recipient] = allowed;
        emit RecipientWhitelistUpdated(recipient, allowed);
    }

    function getBuyPrice(uint256 amount) public view returns (uint256) {
        return _curveIntegral(totalSupply(), amount);
    }

    function getSellPrice(uint256 amount) public view returns (uint256) {
        uint256 supply = totalSupply();
        require(amount <= supply, "amount exceeds supply");
        return _curveIntegral(supply - amount, amount);
    }

    function buyShares(uint256 amount, uint256 maxPrice) external payable nonReentrant {
        uint256 price = getBuyPrice(amount);
        if (price > maxPrice) revert SlippageExceeded();
        if (msg.value < price) revert InsufficientPayment();

        _mint(msg.sender, amount);
        dividendDebt[msg.sender] = _scaledBalance(msg.sender);

        uint256 refund = msg.value - price;
        if (refund > 0) {
            (bool ok,) = msg.sender.call{value: refund}("");
            require(ok, "refund failed");
        }

        emit SharesBought(msg.sender, amount, price);
    }

    function sellShares(uint256 amount, uint256 minPrice) external nonReentrant {
        _claimDividends(msg.sender);

        uint256 price = getSellPrice(amount);
        if (price < minPrice) revert SlippageExceeded();
        _burn(msg.sender, amount);
        dividendDebt[msg.sender] = _scaledBalance(msg.sender);

        (bool ok,) = msg.sender.call{value: price}("");
        require(ok, "payout failed");

        emit SharesSold(msg.sender, amount, price);
    }

    function paySponsored(string calldata) external payable {
        _distributeRevenue(msg.value);
    }

    function subscribe() external payable {
        _distributeRevenue(msg.value);
    }

    function claimableDividends(address investor) public view returns (uint256) {
        uint256 accumulated = _scaledBalance(investor);
        if (accumulated <= dividendDebt[investor]) return 0;
        return accumulated - dividendDebt[investor];
    }

    function claimDividends() public nonReentrant {
        _claimDividends(msg.sender);
    }

    function _claimDividends(address investor) internal {
        uint256 claimable = claimableDividends(investor);
        dividendDebt[investor] = _scaledBalance(investor);
        if (claimable == 0) return;

        investorPool -= claimable;
        (bool ok,) = investor.call{value: claimable}("");
        require(ok, "claim failed");

        emit DividendsClaimed(investor, claimable);
    }

    function spendOps(address payable recipient, uint256 amount, string calldata reason)
        external
        onlyAgentOwnerOrOrchestrator
        nonReentrant
    {
        if (!whitelistedRecipients[recipient]) revert RecipientNotWhitelisted();
        if (amount > operationalBalance) revert InsufficientOperationalBalance();

        operationalBalance -= amount;
        (bool ok,) = recipient.call{value: amount}("");
        require(ok, "ops spend failed");

        emit OpsSpend(recipient, amount, reason);
    }

    function _distributeRevenue(uint256 amount) internal nonReentrant {
        uint256 opsAmount = (amount * OPERATIONAL_BPS) / BPS_DENOMINATOR;
        uint256 investorAmount = (amount * INVESTOR_BPS) / BPS_DENOMINATOR;
        uint256 platformAmount = amount - opsAmount - investorAmount;

        operationalBalance += opsAmount;
        investorPool += investorAmount;

        uint256 supply = totalSupply();
        if (supply > 0 && investorAmount > 0) {
            accDividendPerShare += (investorAmount * ACC_PRECISION) / supply;
        }

        (bool ok,) = platformWallet.call{value: platformAmount}("");
        require(ok, "platform transfer failed");

        emit RevenueDistributed(amount, OPERATIONAL_BPS / 100, INVESTOR_BPS / 100, PLATFORM_BPS / 100);
    }

    function _curveIntegral(uint256 startSupply, uint256 amount) internal view returns (uint256) {
        if (amount == 0) return 0;
        uint256 endSupply = startSupply + amount;
        uint256 linearSum = ((startSupply + endSupply - 1) * amount) / 2;
        return (basePrice * amount) + (slope * linearSum);
    }

    function _scaledBalance(address investor) internal view returns (uint256) {
        return (balanceOf(investor) * accDividendPerShare) / ACC_PRECISION;
    }
}
