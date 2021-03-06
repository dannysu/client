/* @flow */

import React, {Component} from 'react'
import {Text, Icon} from './index'
import {globalStyles, globalColors} from '../styles/style-guide'
import type {Props} from './back-button'

export default class BackButton extends Component {
  props: Props;

  render () {
    return (
      <div style={styles.container} onClick={this.props.onClick} style={this.props.style}>
        <Icon type='fa-arrow-left' style={styles.icon}/>
        <Text inline type='Body' onClick={() => this.props.onClick()}>Back</Text>
      </div>
    )
  }
}

BackButton.propTypes = {
  onClick: React.PropTypes.func.isRequired,
  style: React.PropTypes.object
}

export const styles = {
  container: {
    ...globalStyles.flexBoxRow,
    alignItems: 'center'
  },
  icon: {
    ...globalStyles.clickable,
    marginRight: 10
  }
}

