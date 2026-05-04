// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {AgentINFT} from "../src/AgentINFT.sol";
import {AgentTreasury} from "../src/AgentTreasury.sol";
import {AgentTreasuryFactory} from "../src/AgentTreasuryFactory.sol";

contract AgentContractsTest is Test {
    AgentTreasuryFactory internal factory;
    AgentINFT internal inft;

    address internal architect = address(0xA11CE);
    address internal investor = address(0xB0B);
    address internal brand = address(0xB4A);
    address internal platform = address(0xC0FFEE);
    address internal orchestrator = address(0xDAD);
    address payable internal computeProvider = payable(address(0xFEE));

    uint256 internal constant MINT_FEE = 0.1 ether;
    uint256 internal constant BASE_PRICE = 0.01 ether;
    uint256 internal constant SLOPE = 0.001 ether;

    function setUp() public {
        factory = new AgentTreasuryFactory(platform, BASE_PRICE, SLOPE);
        inft = new AgentINFT(MINT_FEE, orchestrator, factory);
        factory.setRegistry(address(inft));

        vm.deal(architect, 10 ether);
        vm.deal(investor, 10 ether);
        vm.deal(brand, 10 ether);
    }

    function testMintDeploysTreasuryAndStoresMetadata() public {
        vm.prank(architect);
        (uint256 tokenId, address treasury) = inft.mintAgent{value: MINT_FEE}("zg://agent-1", keccak256("prompt"));

        assertEq(tokenId, 1);
        assertEq(inft.ownerOf(tokenId), architect);
        assertEq(inft.metadataPointer(tokenId), "zg://agent-1");
        assertEq(inft.treasuryOf(tokenId), treasury);
        assertEq(factory.treasuryOf(tokenId), treasury);
        assertEq(AgentTreasury(payable(treasury)).owner(), architect);
    }

    function testMintRejectsInsufficientFee() public {
        vm.prank(architect);
        vm.expectRevert(AgentINFT.InsufficientMintFee.selector);
        inft.mintAgent{value: MINT_FEE - 1}("zg://agent-1", keccak256("prompt"));
    }

    function testMetadataAndProofAuthorization() public {
        vm.prank(architect);
        (uint256 tokenId,) = inft.mintAgent{value: MINT_FEE}("zg://agent-1", keccak256("prompt"));

        vm.prank(investor);
        vm.expectRevert(AgentINFT.Unauthorized.selector);
        inft.setMetadataPointer(tokenId, "zg://bad");

        vm.prank(architect);
        inft.setMetadataPointer(tokenId, "zg://agent-2");
        assertEq(inft.metadataPointer(tokenId), "zg://agent-2");

        bytes memory proof = abi.encode("llama-3-8b", bytes32("input"), bytes32("output"), bytes("tee"));
        vm.prank(investor);
        vm.expectRevert(AgentINFT.Unauthorized.selector);
        inft.submitInferenceProof(tokenId, proof);

        vm.prank(orchestrator);
        inft.submitInferenceProof(tokenId, proof);
        assertEq(inft.latestProof(tokenId), keccak256(proof));
    }

    function testCloneRequiresOwnerOrAuthorizedUsage() public {
        vm.prank(architect);
        (uint256 tokenId,) = inft.mintAgent{value: MINT_FEE}("zg://agent-1", keccak256("prompt"));

        vm.prank(investor);
        vm.expectRevert(AgentINFT.Unauthorized.selector);
        inft.clone{value: MINT_FEE}(tokenId, investor);

        vm.prank(architect);
        inft.authorizeUsage(tokenId, investor, true);

        vm.prank(investor);
        (uint256 cloneId,) = inft.clone{value: MINT_FEE}(tokenId, investor);
        assertEq(inft.ownerOf(cloneId), investor);
        assertEq(inft.metadataPointer(cloneId), "zg://agent-1");
    }

    function testBondingCurveBuySellAndSlippage() public {
        AgentTreasury treasury = _mintTreasury();

        assertEq(treasury.getBuyPrice(3), 0.033 ether);

        vm.prank(investor);
        vm.expectRevert(AgentTreasury.SlippageExceeded.selector);
        treasury.buyShares{value: 0.033 ether}(3, 0.032 ether);

        vm.prank(investor);
        treasury.buyShares{value: 0.033 ether}(3, 0.033 ether);
        assertEq(treasury.balanceOf(investor), 3);
        assertEq(treasury.totalSupply(), 3);
        assertEq(treasury.getSellPrice(2), 0.023 ether);

        vm.prank(investor);
        treasury.sellShares(2, 0.023 ether);
        assertEq(treasury.balanceOf(investor), 1);
    }

    function testRevenueSplitAndDividendClaim() public {
        AgentTreasury treasury = _mintTreasury();

        uint256 buyPrice = treasury.getBuyPrice(10);
        vm.prank(investor);
        treasury.buyShares{value: buyPrice}(10, buyPrice);

        uint256 platformBefore = platform.balance;
        vm.prank(brand);
        treasury.paySponsored{value: 1 ether}("sponsored:demo");

        assertEq(treasury.operationalBalance(), 0.7 ether);
        assertEq(treasury.investorPool(), 0.2 ether);
        assertEq(platform.balance - platformBefore, 0.1 ether);
        assertEq(treasury.claimableDividends(investor), 0.2 ether);

        uint256 investorBefore = investor.balance;
        vm.prank(investor);
        treasury.claimDividends();
        assertEq(investor.balance - investorBefore, 0.2 ether);
        assertEq(treasury.investorPool(), 0);
    }

    function testSpendOpsRequiresWhitelistedRecipient() public {
        AgentTreasury treasury = _mintTreasury();

        vm.prank(brand);
        treasury.subscribe{value: 1 ether}();

        vm.prank(orchestrator);
        vm.expectRevert(AgentTreasury.RecipientNotWhitelisted.selector);
        treasury.spendOps(computeProvider, 0.1 ether, "compute");

        vm.prank(architect);
        treasury.setWhitelistedRecipient(computeProvider, true);

        uint256 providerBefore = computeProvider.balance;
        vm.prank(orchestrator);
        treasury.spendOps(computeProvider, 0.1 ether, "compute");

        assertEq(computeProvider.balance - providerBefore, 0.1 ether);
        assertEq(treasury.operationalBalance(), 0.6 ether);
    }

    function _mintTreasury() internal returns (AgentTreasury) {
        vm.prank(architect);
        (uint256 tokenId, address treasury) = inft.mintAgent{value: MINT_FEE}("zg://agent-1", keccak256("prompt"));
        assertEq(tokenId, 1);
        return AgentTreasury(payable(treasury));
    }
}
