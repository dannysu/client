'use strict'
/**
 * @providesModule Meta Navigator
 * A Meta navigator for handling different navigators at the top level.
 * todo(mm) explain why we need a meta navigator
 */

import React, { Component, Navigator, View } from 'react-native'
import { connect } from 'react-redux/native'
import Immutable from 'immutable'

class MetaNavigator extends Component {
  constructor () {
    super()

    this.state = {
    }
  }

  isParentOfRoute (routeParent, routeMaybeChild) {
    return (
      !Immutable.is(routeMaybeChild, routeParent) &&
      Immutable.is(routeMaybeChild.slice(0, routeParent.count()), routeParent)
    )
  }

  shouldComponentUpdate (nextProps, nextState) {
    const { store, rootRouteParser } = this.props
    const route = this.props.uri
    const nextRoute = nextProps.uri

    const { componentAtTop, routeStack: nextRouteStack } = this.getComponentAtTop(rootRouteParser, store, nextRoute)
    if (nextProps === this.props && nextState === this.state) {
      return false
    } else if (this.isParentOfRoute(route, nextRoute)) {
      this.refs.navigator.push(componentAtTop)
      return true
    // TODO: also check to see if this route exists in the navigator's route
    } else if (this.isParentOfRoute(nextRoute, route)) {
      const navRoutes = this.refs.navigator.getCurrentRoutes()
      const targetRoute = navRoutes.reverse().find(navRoute =>
          navRoute.component === componentAtTop.component && navRoute.title === componentAtTop.title
      )
      this.refs.navigator.popToRoute(targetRoute)
      return true
    } else {
      this.refs.navigator.immediatelyResetRouteStack(nextRouteStack.toJS())
      return true
    }
  }

  componentDidMount () {
    // TODO FIX this...
    // This is just to fix an error we get from the navigator complaining about
    // some var elgibleGestures not setup. This hack sets it up.
    this.refs.navigator._handleTouchStart()
  }

  findGlobalRouteHandler (currentPath) {
    let parseRoute = null
    if (this.props.globalRoutes) {
      this.props.globalRoutes.forEach((route) => {
        if (route.canParseNextRoute(currentPath)) {
          parseRoute = route.parseRoute
          return false // short circuit
        } else {
          return true
        }
      })
    }

    return parseRoute
  }

  getComponentAtTop (rootRouteParser, store, uri) {
    let currentPath = uri.first() || Immutable.Map()
    let nextPath = uri.rest().first() || Immutable.Map()
    let restPath = uri.rest().rest()
    let routeStack = Immutable.List()

    let parseNextRoute = rootRouteParser
    let componentAtTop = null

    while (parseNextRoute) {
      const t = parseNextRoute(store, currentPath, nextPath, uri)
      componentAtTop = t.componentAtTop
      parseNextRoute = t.parseNextRoute
      routeStack = routeStack.push(componentAtTop)

      currentPath = nextPath
      nextPath = restPath.first() || Immutable.Map()
      restPath = restPath.rest()

      if (!parseNextRoute) {
        parseNextRoute = this.findGlobalRouteHandler(currentPath)
      }
    }

    return {componentAtTop, routeStack}
  }

  render () {
    // TODO (mm): know when to create a new navigator

    // TODO (mm): specify the prop types
    const { store, rootRouteParser, uri, NavBar } = this.props

    let {componentAtTop, routeStack} = this.getComponentAtTop(rootRouteParser, store, uri)

    return (
      <Navigator
        saveName='main'
        ref='navigator'
        initialRouteStack={routeStack.toJS()}
        renderScene={(route, navigator) => {
          return (
            <View style={{flex: 1, marginTop: route.hideNavBar ? 0 : this.props.navBarHeight}}>
              {React.createElement(connect(componentAtTop.mapStateToProps || (state => state))(route.component), {...route.props})}
            </View>
          )
        }}
        navigationBar={componentAtTop.hideNavBar ? null : NavBar}
      />
    )
  }
}

MetaNavigator.propTypes = {
  uri: React.PropTypes.object.isRequired,
  store: React.PropTypes.object.isRequired,
  NavBar: React.PropTypes.object.isRequired,
  rootRouteParser: React.PropTypes.func.isRequired,
  globalRoutes: React.PropTypes.object,
  navBarHeight: React.PropTypes.number.isRequired
}

export default MetaNavigator