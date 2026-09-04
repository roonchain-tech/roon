const { expect } = require('chai')
const hre = require('hardhat')
const { ethers } = hre
const { findEvent, waitWithTimeout, RETRY_DELAY_FUNC, getValidatorHexAddresses, hexToBech32Addr} = require('../common')

describe('Distribution – withdraw validator commission', function () {
    const DIST_ADDRESS = '0x0000000000000000000000000000000000000801'
    const GAS_LIMIT    = 1_000_000

    let distribution, validator

    before(async () => {
        const signers   = await ethers.getSigners()
        // Commission withdrawal must be signed by the validator operator.
        // The signer (accounts[0]) is the operator of the validator created
        // in 1_create_and_edit_validator.js, which accrues commission from
        // its self-delegation.
        validator       = signers[0]
        distribution    = await ethers.getContractAt('DistributionI', DIST_ADDRESS)
    })

    it('withdraws validator commission and emits proper event', async function () {
        // Resolve the signer's validator at runtime (created in 1_create_and_edit_validator.js)
        const valHexAddrs = await getValidatorHexAddresses(hre)
        const valHex = valHexAddrs.find(a => a.toLowerCase() === validator.address.toLowerCase())
        const valBech32 = await hexToBech32Addr(hre, valHex, 'roonvaloper')

        // 1) query commission before withdrawal
        const beforeRes = await distribution.validatorCommission(valBech32)
        const beforeAmt = beforeRes.length
            ? BigInt(beforeRes[0].amount.toString())
            : 0n

        // 2) withdraw commission
        const tx      = await distribution
            .connect(validator)
            .withdrawValidatorCommission(valBech32, { gasLimit: GAS_LIMIT })
        const receipt = await waitWithTimeout(tx, 20000, RETRY_DELAY_FUNC)

        // 3) parse the event
        const parsedEvt = findEvent(receipt.logs, distribution.interface, 'WithdrawValidatorCommission')
        expect(parsedEvt, 'event must be emitted').to.exist

        // 4) verify the indexed validatorAddress via topic hash
        const rawLog      = receipt.logs.find(log => {
            try { return distribution.interface.parseLog(log).name === 'WithdrawValidatorCommission' }
            catch { return false }
        })
        const expectedTopic = ethers.keccak256(ethers.toUtf8Bytes(valBech32))
        expect(rawLog.topics[1]).to.equal(expectedTopic)

        // 5) verify commission amount in event ≥ beforeAmt
        const commissionBn = parsedEvt.args.commission
        const commission   = BigInt(commissionBn.toString())
        expect(commission).to.be.gte(beforeAmt)

        // 6) query commission after withdrawal
        const afterRes = await distribution.validatorCommission(valBech32)
        const afterAmt = afterRes.length
            ? BigInt(afterRes[0].amount.toString())
            : 0n

        expect(afterAmt).to.be.lessThan(beforeAmt, 'Commission should be reduced')
    })
})