'use strict'
/* @flow */

export default function (props, state) {
  console.log('inside login container')
  console.log(props)
  console.log(state)
  return (<div><p>login container {props.foo}</p></div>)
}