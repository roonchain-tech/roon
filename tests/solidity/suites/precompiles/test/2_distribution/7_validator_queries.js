const { expect } = require('chai');
const hre = require('hardhat');
const { getValidatorHexAddresses, hexToBech32Addr } = require('../common');

describe('Distribution – validator query methods', function () {
    const DIST_ADDRESS = '0x0000000000000000000000000000000000000801';

    let distribution, signer;
    let VAL_OPER_BECH32, VAL_BECH32;

    before(async () => {
        [signer] = await hre.ethers.getSigners();
        distribution = await hre.ethers.getContractAt('DistributionI', DIST_ADDRESS);
        // Resolve the signer's validator at runtime (created in
        // 1_create_and_edit_validator.js); the genesis validator key is random.
        const valHexAddrs = await getValidatorHexAddresses(hre);
        const signerValHex = valHexAddrs.find(a => a.toLowerCase() === signer.address.toLowerCase()) || valHexAddrs[0];
        VAL_OPER_BECH32 = await hexToBech32Addr(hre, signerValHex, 'roonvaloper');
        VAL_BECH32 = await hexToBech32Addr(hre, signerValHex, 'roon');
    });

    it('validatorDistributionInfo returns current distribution info', async function () {
        const info = await distribution.validatorDistributionInfo(VAL_OPER_BECH32);
        console.log('validatorDistributionInfo:', info);
        expect(info.operatorAddress).to.equal(VAL_BECH32);
        expect(info.selfBondRewards).to.be.an('array');
        expect(info.commission).to.be.an('array');
    });

    it('validatorSlashes returns slashing events (none expected)', async function () {
        const pageReq = { key: '0x', offset: 0, limit: 100, countTotal: true, reverse: false };
        const [slashes, pageResponse] = await distribution.validatorSlashes(
            VAL_OPER_BECH32,
            1,
            5,
            pageReq
        );
        console.log('validatorSlashes:', slashes, pageResponse);
        expect(slashes).to.be.an('array');
        expect(slashes.length).to.equal(Number(pageResponse.total.toString()));
    });

    it('delegatorValidators lists validators for delegator', async function () {
        const validators = await distribution.delegatorValidators(signer.address);
        console.log('delegatorValidators:', validators);
        console.log(validators)
        expect(validators).to.include(VAL_OPER_BECH32);
    });
});