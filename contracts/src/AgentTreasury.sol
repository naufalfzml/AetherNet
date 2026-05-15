// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @title AetherNet Agent Treasury
/// @notice Holds an agent's investment market, operational balance, and investor dividend pool.
/// @dev Shares are non-transferable and can only be minted or burned through the bonding-curve entrypoints.
contract AgentTreasury is ERC20, Ownable, ReentrancyGuard {
    /// @notice Raised when a quoted buy or sell price violates the caller's slippage bound.
    error SlippageExceeded();
    /// @notice Raised when a buyer does not send enough native token for a share purchase.
    error InsufficientPayment();
    /// @notice Raised when curve reserve cannot satisfy a share sale payout.
    error InsufficientCurveReserve();
    /// @notice Raised when operational balance cannot satisfy an ops spend.
    error InsufficientOperationalBalance();
    /// @notice Raised when an ops recipient has not been explicitly whitelisted.
    error RecipientNotWhitelisted();
    /// @notice Raised when attempting to transfer shares peer-to-peer.
    error SharesNonTransferable();
    /// @notice Raised when the caller lacks the required role.
    error Unauthorized();

    /// @notice Emitted when a buyer acquires shares from the bonding curve.
    /// @param buyer Wallet that purchased shares.
    /// @param amount Number of shares purchased.
    /// @param paid Native-token amount paid for the purchase.
    event SharesBought(address indexed buyer, uint256 amount, uint256 paid);
    /// @notice Emitted when a holder sells shares back into the bonding curve.
    /// @param seller Wallet that sold shares.
    /// @param amount Number of shares sold.
    /// @param received Native-token amount received from the sale.
    event SharesSold(address indexed seller, uint256 amount, uint256 received);
    /// @notice Emitted when revenue is split across ops, investors, and platform.
    /// @param amount Gross native-token revenue distributed.
    /// @param operationalBps Operational share expressed in percentage points.
    /// @param investorBps Investor share expressed in percentage points.
    /// @param platformBps Platform share expressed in percentage points.
    event RevenueDistributed(uint256 amount, uint256 operationalBps, uint256 investorBps, uint256 platformBps);
    /// @notice Emitted when an investor claims accrued dividends.
    /// @param investor Wallet receiving the dividend payout.
    /// @param amount Native-token amount claimed.
    event DividendsClaimed(address indexed investor, uint256 amount);
    /// @notice Emitted when operational funds are spent from the treasury.
    /// @param recipient Recipient of the ops payment.
    /// @param amount Native-token amount spent.
    /// @param reason Freeform spend reason for offchain tracing.
    event OpsSpend(address indexed recipient, uint256 amount, string reason);
    /// @notice Emitted when an ops recipient whitelist entry changes.
    /// @param recipient Address whose whitelist status changed.
    /// @param allowed Whether the recipient is now allowed.
    event RecipientWhitelistUpdated(address indexed recipient, bool allowed);
    /// @notice Emitted when the treasury orchestrator changes.
    /// @param orchestrator New orchestrator address.
    event OrchestratorUpdated(address indexed orchestrator);

    /// @notice Basis-point denominator used for revenue splits.
    uint256 public constant BPS_DENOMINATOR = 10_000;
    /// @notice Operational share of gross revenue in basis points.
    uint256 public constant OPERATIONAL_BPS = 7_000;
    /// @notice Investor share of gross revenue in basis points.
    uint256 public constant INVESTOR_BPS = 2_000;
    /// @notice Platform share of gross revenue in basis points.
    uint256 public constant PLATFORM_BPS = 1_000;
    uint256 private constant ACC_PRECISION = 1e36;

    /// @notice Agent token id that owns this treasury.
    uint256 public immutable tokenId;
    /// @notice Starting marginal share price before any supply exists.
    uint256 public immutable basePrice;
    /// @notice Linear slope applied to marginal share pricing.
    uint256 public immutable slope;
    /// @notice Platform wallet that receives the platform revenue split.
    address public immutable platformWallet;
    /// @notice Registry allowed to synchronize orchestrator changes.
    address public immutable registry;

    /// @notice Address allowed to spend ops and coordinate runtime actions.
    address public orchestrator;
    /// @notice Native-token reserve backing share exit liquidity.
    uint256 public curveReserve;
    /// @notice Native-token balance reserved for runtime operations.
    uint256 public operationalBalance;
    /// @notice Native-token balance reserved for investor dividend payouts.
    uint256 public investorPool;
    /// @notice Accumulated dividends per share, scaled by ACC_PRECISION.
    uint256 public accDividendPerShare;

    /// @notice Tracks which addresses may receive ops spend transfers.
    mapping(address => bool) public whitelistedRecipients;
    /// @notice Tracks claimed dividend accumulator checkpoints per investor.
    mapping(address => uint256) public dividendDebt;

    /// @notice Restricts access to the agent owner or treasury orchestrator.
    modifier onlyAgentOwnerOrOrchestrator() {
        if (msg.sender != owner() && msg.sender != orchestrator) revert Unauthorized();
        _;
    }

    /// @param tokenId_ Agent token id tied to this treasury.
    /// @param agentOwner Owner address for the agent and treasury.
    /// @param platformWallet_ Platform revenue recipient.
    /// @param registry_ Agent registry allowed to synchronize orchestrator rotation.
    /// @param orchestrator_ Initial orchestrator for runtime operations.
    /// @param basePrice_ Starting marginal share price.
    /// @param slope_ Linear share-price slope.
    constructor(
        uint256 tokenId_,
        address agentOwner,
        address platformWallet_,
        address registry_,
        address orchestrator_,
        uint256 basePrice_,
        uint256 slope_
    ) ERC20("AetherNet Agent Share", "AAS") Ownable(agentOwner) {
        tokenId = tokenId_;
        platformWallet = platformWallet_;
        registry = registry_;
        orchestrator = orchestrator_;
        basePrice = basePrice_;
        slope = slope_;
    }

    /// @notice Accepts direct native-token top-ups into the operational balance.
    receive() external payable {
        operationalBalance += msg.value;
    }

    /// @notice Returns whole-number share decimals for the investment token.
    /// @return Decimal count, fixed at zero.
    function decimals() public pure override returns (uint8) {
        return 0;
    }

    /// @notice Updates the treasury orchestrator.
    /// @param nextOrchestrator New orchestrator address.
    function setOrchestrator(address nextOrchestrator) external onlyOwner {
        orchestrator = nextOrchestrator;
        emit OrchestratorUpdated(nextOrchestrator);
    }

    /// @notice Synchronizes the treasury orchestrator from the registry.
    /// @param nextOrchestrator New orchestrator address.
    function syncOrchestrator(address nextOrchestrator) external {
        if (msg.sender != registry) revert Unauthorized();
        orchestrator = nextOrchestrator;
        emit OrchestratorUpdated(nextOrchestrator);
    }

    /// @notice Updates whether an address may receive ops spend transfers.
    /// @param recipient Address to update.
    /// @param allowed Whether the recipient should be allowed.
    function setWhitelistedRecipient(address recipient, bool allowed) external onlyOwner {
        whitelistedRecipients[recipient] = allowed;
        emit RecipientWhitelistUpdated(recipient, allowed);
    }

    /// @notice Quotes the total native-token cost to buy a number of shares.
    /// @param amount Number of shares to purchase.
    /// @return Total native-token cost across the linear bonding curve.
    function getBuyPrice(uint256 amount) public view returns (uint256) {
        return _curveIntegral(totalSupply(), amount);
    }

    /// @notice Quotes the total native-token return to sell a number of shares.
    /// @param amount Number of shares to sell.
    /// @return Total native-token payout across the linear bonding curve.
    function getSellPrice(uint256 amount) public view returns (uint256) {
        uint256 supply = totalSupply();
        require(amount <= supply, "amount exceeds supply");
        return _curveIntegral(supply - amount, amount);
    }

    /// @notice Buys shares from the bonding curve and adds payment into curve reserve.
    /// @param amount Number of shares to buy.
    /// @param maxPrice Maximum acceptable total purchase price.
    function buyShares(uint256 amount, uint256 maxPrice) external payable nonReentrant {
        uint256 price = getBuyPrice(amount);
        if (price > maxPrice) revert SlippageExceeded();
        if (msg.value < price) revert InsufficientPayment();

        _claimDividends(msg.sender);
        _mint(msg.sender, amount);
        dividendDebt[msg.sender] = _scaledBalance(msg.sender);
        curveReserve += price;

        uint256 refund = msg.value - price;
        if (refund > 0) {
            (bool ok,) = msg.sender.call{value: refund}("");
            require(ok, "refund failed");
        }

        emit SharesBought(msg.sender, amount, price);
    }

    /// @notice Sells shares back to the bonding curve and withdraws from curve reserve.
    /// @param amount Number of shares to sell.
    /// @param minPrice Minimum acceptable total sale price.
    function sellShares(uint256 amount, uint256 minPrice) external nonReentrant {
        _claimDividends(msg.sender);

        uint256 price = getSellPrice(amount);
        if (price < minPrice) revert SlippageExceeded();
        if (price > curveReserve) revert InsufficientCurveReserve();
        _burn(msg.sender, amount);
        dividendDebt[msg.sender] = _scaledBalance(msg.sender);
        curveReserve -= price;

        (bool ok,) = msg.sender.call{value: price}("");
        require(ok, "payout failed");

        emit SharesSold(msg.sender, amount, price);
    }

    /// @notice Distributes sponsored-post revenue across treasury buckets and platform wallet.
    /// @param campaignRef Freeform sponsorship metadata retained only at the calldata layer.
    function paySponsored(string calldata campaignRef) external payable nonReentrant {
        campaignRef;
        _distributeRevenue(msg.value);
    }

    /// @notice Distributes subscription revenue across treasury buckets and platform wallet.
    function subscribe() external payable nonReentrant {
        _distributeRevenue(msg.value);
    }

    /// @notice Returns the currently claimable dividend amount for an investor.
    /// @param investor Investor wallet to inspect.
    /// @return Native-token dividend amount available to claim.
    function claimableDividends(address investor) public view returns (uint256) {
        uint256 accumulated = _scaledBalance(investor);
        if (accumulated <= dividendDebt[investor]) return 0;
        return accumulated - dividendDebt[investor];
    }

    /// @notice Claims accrued dividends for the caller.
    function claimDividends() public nonReentrant {
        _claimDividends(msg.sender);
    }

    /// @notice Claims accrued dividends for an investor and updates its debt checkpoint.
    /// @param investor Investor wallet that will receive dividends.
    function _claimDividends(address investor) internal {
        uint256 claimable = claimableDividends(investor);
        dividendDebt[investor] = _scaledBalance(investor);
        if (claimable == 0) return;

        investorPool -= claimable;
        (bool ok,) = investor.call{value: claimable}("");
        require(ok, "claim failed");

        emit DividendsClaimed(investor, claimable);
    }

    /// @notice Spends operational funds to an approved provider.
    /// @param recipient Recipient of the ops payment.
    /// @param amount Native-token amount to spend.
    /// @param reason Freeform reason emitted for offchain tracing.
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

    /// @notice Splits incoming revenue into ops, investor, and platform buckets.
    /// @param amount Gross native-token amount to distribute.
    function _distributeRevenue(uint256 amount) internal {
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

    /// @notice Integrates a linear bonding curve over a share interval.
    /// @param startSupply Supply level at which integration starts.
    /// @param amount Number of shares being integrated.
    /// @return Total native-token amount across the integrated interval.
    function _curveIntegral(uint256 startSupply, uint256 amount) internal view returns (uint256) {
        if (amount == 0) return 0;
        uint256 endSupply = startSupply + amount;
        uint256 linearSum = ((startSupply + endSupply - 1) * amount) / 2;
        return (basePrice * amount) + (slope * linearSum);
    }

    /// @notice Returns the investor's accumulated dividend checkpoint in native-token units.
    /// @param investor Investor wallet to inspect.
    /// @return Accumulated dividend amount attributable to the investor balance.
    function _scaledBalance(address investor) internal view returns (uint256) {
        return (balanceOf(investor) * accDividendPerShare) / ACC_PRECISION;
    }

    /// @notice Prevents peer-to-peer transfers while allowing mint and burn flows.
    /// @param from Sender address in the token update lifecycle.
    /// @param to Recipient address in the token update lifecycle.
    /// @param value Token amount being updated.
    function _update(address from, address to, uint256 value) internal override {
        if (from != address(0) && to != address(0)) revert SharesNonTransferable();
        super._update(from, to, value);
    }
}
