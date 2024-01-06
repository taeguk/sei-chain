const { expect } = require("chai");
const {isBigNumber} = require("hardhat/common");
const { exec } = require("child_process"); // Importing exec from child_process


async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function delay() {
  await sleep(3000)
}

function debug(msg) {
  console.log(msg)
}

const CW20_BASE_WASM_LOCATION = "../cw20_base.wasm";

describe("CW20ERC20WrapperTest", function () {
    let adminAddr;
    let contractAddress;
    let deployerAddr;
    let cW20ERC20Wrapper;

    before(async function () {
        let codeId = await deployWasm();
        console.log(`codeId: ${codeId}`);
        adminAddr = await getAdmin();
        console.log(`adminAddr: ${adminAddr}`);
        contractAddress = await instantiateWasm(codeId, adminAddr);
        console.log(`contractAddress: ${contractAddress}`)
        // deploy the CW20ERC20Wrapper solidity contract with the contractAddress passed in
        const [deployer] = await hre.ethers.getSigners();
        await fundDeployer(deployer.address);
    
        console.log(
            "Deploying contracts with the account:",
            deployer.address
        );
        await delay();
        const CW20ERC20Wrapper = await ethers.getContractFactory("CW20ERC20Wrapper");
        await delay();
        cW20ERC20Wrapper = await CW20ERC20Wrapper.deploy(contractAddress, "BTOK", "TOK");
        await cW20ERC20Wrapper.waitForDeployment();
        console.log("CW20ERC20Wrapper address = ", cW20ERC20Wrapper.target)
        deployerAddr = await deployer.getAddress();
    });

    describe("balanceOf", function () {
        it("balanceOf should work", async function () {
            let addressToCheck = deployerAddr;
            console.log(`addressToCheck: ${addressToCheck}`)
            let balance = await cW20ERC20Wrapper.balanceOf(addressToCheck);
            console.log(`Balance of ${addressToCheck}: ${balance}`); // without this line the test fails more frequently
            expect(Number(balance)).to.equal(1000000);
        });
    });

    describe("totalSupply", function () {
        it("totalSupply should work", async function () {
            let totalSupply = await cW20ERC20Wrapper.totalSupply();
            console.log(`Total supply: ${totalSupply}`);
            expect(Number(totalSupply)).to.equal(1000000);
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
            console.log(`stdout: ${stdout}`)
            resolve(stdout.trim());
        });
    });
    return adminAddr;
}

async function instantiateWasm(codeId, adminAddr) {
    // Wrap the exec function in a Promise
    let contractAddress = await new Promise((resolve, reject) => {
        exec(`seid tx wasm instantiate ${codeId} '{ "name": "BTOK", "symbol": "BTOK", "decimals": 6, "initial_balances": [ { "address": "${adminAddr}", "amount": "1000000" } ], "mint": { "minter": "${adminAddr}", "cap": "99900000000" } }' --label cw20-test --admin ${adminAddr} --from admin --gas=5000000 --fees=1000000usei -y --broadcast-mode block`, (error, stdout, stderr) => {
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