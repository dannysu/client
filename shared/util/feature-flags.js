/* @flow */

type FeatureFlag = {
  tracker: 'v1' | 'v2',
  allowLogin: boolean
}

const tracker = process.env.KEYBASE_TRACKER_V2 ? 'v2' : 'v1'
const allowLogin = !!process.env.KEYBASE_ALLOW_LOGIN

const ff: FeatureFlag = {
  tracker,
  allowLogin
}

export default ff
export {tracker, allowLogin}
