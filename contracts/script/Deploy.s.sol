// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script} from "forge-std/Script.sol";
import {AgentINFT} from "../src/AgentINFT.sol";
import {AgentTreasuryFactory} from "../src/AgentTreasuryFactory.sol";

contract Deploy is Script {
    function run() external returns (AgentTreasuryFactory factory, AgentINFT inft) {
        uint256 deployerKey = vm.envUint("PRIVATE_KEY");
        address platformWallet = vm.envOr("PLATFORM_WALLET", vm.addr(deployerKey));
        address orchestrator = vm.envOr("ORCHESTRATOR_ADDRESS", vm.addr(deployerKey));
        uint256 mintFee = vm.envOr("MINT_FEE_WEI", uint256(0.005 ether));
        uint256 basePrice = vm.envOr("BASE_SHARE_PRICE_WEI", uint256(0.001 ether));
        uint256 slope = vm.envOr("SHARE_SLOPE_WEI", uint256(0.0001 ether));

        vm.startBroadcast(deployerKey);
        factory = new AgentTreasuryFactory(platformWallet, basePrice, slope);
        inft = new AgentINFT(mintFee, orchestrator, factory);
        factory.setRegistry(address(inft));
        vm.stopBroadcast();

        string memory root = "deployment";
        vm.serializeAddress(root, "agentTreasuryFactory", address(factory));
        vm.serializeAddress(root, "agentINFT", address(inft));
        vm.serializeAddress(root, "platformWallet", platformWallet);
        vm.serializeAddress(root, "orchestrator", orchestrator);
        vm.serializeUint(root, "chainId", block.chainid);
        vm.serializeUint(root, "mintFeeWei", mintFee);
        vm.serializeUint(root, "baseSharePriceWei", basePrice);
        string memory json = vm.serializeUint(root, "shareSlopeWei", slope);
        vm.writeJson(json, "deployments/0g-testnet.json");
    }
}
