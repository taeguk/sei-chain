const { expect } = require("chai");
const {isBigNumber} = require("hardhat/common");
const { exec } = require("child_process"); // Importing exec from child_process
const { cons } = require("fp-ts/lib/NonEmptyArray2v");


async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function delay() {
  await sleep(3000)
}

function debug(msg) {
  //console.log(msg)
}

const CW20_BASE_WASM_LOCATION = "../cw20_base.wasm";

describe("CW20ERC20WrapperTest", function () {
    let adminAddrSei;
    let contractAddress;
    let deployerAddr;
    let cW20ERC20Wrapper;
    let secondAnvilAddrETH = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8";

    before(async function () {
        console.log("deploying wasm...")
        let codeId = await deployWasm();
        console.log(`codeId: ${codeId}`);
        console.log("getting admin addr...")
        adminAddrSei = await getAdmin();
        console.log(`adminAddr: ${adminAddrSei}`);
        console.log("instantiating wasm...")
        contractAddress = await instantiateWasm(codeId, adminAddrSei);
        console.log(`contractAddress: ${contractAddress}`)
        // deploy the CW20ERC20Wrapper solidity contract with the contractAddress passed in
        // let signers = await ethers.getSigners();
        // owner = signers[0]
        let signers = await hre.ethers.getSigners();
        const deployer = signers[0]
        deployerAddr = await deployer.getAddress()
        console.log("deployerAddr = ", deployerAddr);
        await fundDeployer(deployerAddr);
    
        console.log(
            "Deploying contracts with the account:",
           deployerAddr 
        );
        await delay();
        const CW20ERC20Wrapper = await ethers.getContractFactory("CW20ERC20Wrapper");
        await delay();
        console.log("deploying cw20 erc20 wrapper...")
        cW20ERC20Wrapper = await CW20ERC20Wrapper.deploy(contractAddress, "BTOK", "TOK");
        await cW20ERC20Wrapper.waitForDeployment();
        console.log("CW20ERC20Wrapper address = ", cW20ERC20Wrapper.target)
    });

    describe("balanceOf", function () {
        it("balanceOf should work", async function () {
            let addressToCheck = secondAnvilAddrETH;
            console.log(`addressToCheck: ${addressToCheck}`);
            let secondAnvilAddrBalance = await cW20ERC20Wrapper.balanceOf(addressToCheck);
            console.log(`Balance of ${addressToCheck}: ${secondAnvilAddrBalance}`); // without this line the test fails more frequently
            expect(Number(secondAnvilAddrBalance)).to.be.greaterThan(0);
        });
    });

    describe("totalSupply", function () {
        it("totalSupply should work", async function () {
            let totalSupply = await cW20ERC20Wrapper.totalSupply();
            console.log(`Total supply: ${totalSupply}`);
            // expect total supply to be great than 0
            expect(Number(totalSupply)).to.be.greaterThan(0);
        });
    });

    describe("allowance", function () {
        it("allowance should work", async function () {
            let owner = deployerAddr; // Replace with the owner's address
            let spender = deployerAddr; // Replace with the spender's address
            let allowance = await cW20ERC20Wrapper.allowance(owner, spender);
            console.log(`Allowance for ${spender} from ${owner}: ${allowance}`);
            expect(Number(allowance)).to.equal(0); // Replace with the expected allowance
        });
    });

    describe("approve", function () {
        it("approve should work", async function () {
            let spender = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; // just want a random address that is not deployer
            let amount = 1000000; // Replace with the amount to approve
            console.log("deployerAddr = ", deployerAddr);
            console.log("spender = ", spender);
            const tx = await cW20ERC20Wrapper.approve(spender, amount);
            await tx.wait();
            let allowance = await cW20ERC20Wrapper.allowance(deployerAddr, spender);
            await delay();
            console.log(`Allowance for ${spender} from ${deployerAddr}: ${allowance}`);
            expect(Number(allowance)).to.equal(amount);
        });
    });

    describe("transfer", function () {
        it("transfer should work", async function () {
            let transferReq = await cW20ERC20Wrapper.transferReq("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", 1000000);
            console.log("transferReq = ", transferReq);
            let recipient = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; // Replace with the recipient's address
            let amount = 1000000; // Replace with the amount to transfer
            console.log("deployerAddr = ", deployerAddr);
            console.log("recipient = ", recipient);
            const tx = await cW20ERC20Wrapper.transfer(recipient, amount);
            const receipt = await tx.wait();
            console.log("receipt = ", receipt);
            let balance = await cW20ERC20Wrapper.balanceOf(recipient);
            await delay();
            console.log(`Balance of ${recipient}: ${balance}`);
            expect(Number(balance)).to.equal(amount);
        });
    });
});

async function fundDeployer(deployerAddress) {
    // Wrap the exec function in a Promise
    await new Promise((resolve, reject) => {
        exec(`seid tx evm send ${deployerAddress} 10000000000000000000 --from admin`, (error, stdout, stderr) => {
            if (error) {
                console.log(`error: ${error.message}`);
                reject(error);
                return;
            }
            if (stderr) {
                console.log(`stderr: ${stderr}`);
                reject(new Error(stderr));
                return;
            }
            debug(`stdout: ${stdout}`)
            resolve();
        });
    });
}



async function deployWasm() {
    // Wrap the exec function in a Promise
    let codeId = await new Promise((resolve, reject) => {
        exec(`seid tx wasm store ${CW20_BASE_WASM_LOCATION} --from admin --gas=5000000 --fees=1000000usei -y --broadcast-mode block`, (error, stdout, stderr) => {
            if (error) {
                console.log(`error: ${error.message}`);
                reject(error);
                return;
            }
            if (stderr) {
                console.log(`stderr: ${stderr}`);
                reject(new Error(stderr));
                return;
            }
            debug(`stdout: ${stdout}`)

            // Regular expression to find the 'code_id' value
            const regex = /key: code_id\s+value: "(\d+)"/;

            // Searching for the pattern in the string
            const match = stdout.match(regex);

            let cId = null;
            if (match && match[1]) {
                // The captured group is the code_id value
                cId = match[1];
            }

            console.log(`cId: ${cId}`);
            resolve(cId);
        });
    });

    return codeId;
}

async function getAdmin() {
    // Wrap the exec function in a Promise
    let adminAddr = await new Promise((resolve, reject) => {
        exec(`seid keys show admin -a`, (error, stdout, stderr) => {
            if (error) {
                console.log(`error: ${error.message}`);
                reject(error);
                return;
            }
            if (stderr) {
                console.log(`stderr: ${stderr}`);
                reject(new Error(stderr));
                return;
            }
            debug(`stdout: ${stdout}`)
            resolve(stdout.trim());
        });
    });
    return adminAddr;
}

async function instantiateWasm(codeId, adminAddr) {
    // Wrap the exec function in a Promise
    let secondAnvilAddr = "sei1cjzphr67dug28rw9ueewrqllmxlqe5f0awulvy"; // 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 (known pk)
    console.log("instantiateWasm: will fund admin addr = ", adminAddr);
    console.log("instantiateWasm: will fund secondAnvilAddr = ", secondAnvilAddr);
    let contractAddress = await new Promise((resolve, reject) => {
        const cmd = `seid tx wasm instantiate ${codeId} '{ "name": "BTOK", "symbol": "BTOK", "decimals": 6, "initial_balances": [ { "address": "${adminAddr}", "amount": "1000000" }, { "address": "${secondAnvilAddr}", "amount": "1000000"} ], "mint": { "minter": "${adminAddr}", "cap": "99900000000" } }' --label cw20-test --admin ${adminAddr} --from admin --gas=5000000 --fees=1000000usei -y --broadcast-mode block`;
        exec(cmd, (error, stdout, stderr) => {
            if (error) {
                console.log(`error: ${error.message}`);
                reject(error);
                return;
            }
            if (stderr) {
                console.log(`stderr: ${stderr}`);
                reject(new Error(stderr));
                return;
            }
            debug(`stdout: ${stdout}`)
            const regex = /_contract_address\s*value:\s*(\w+)/;
            const match = stdout.match(regex);
            if (match && match[1]) {
                resolve(match[1]);
            } else {
                reject(new Error('Contract address not found'));
            }
        });
    });
    return contractAddress;
}