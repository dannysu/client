import Window from './window'
import {app, ipcMain} from 'electron'
import menuHelper from './menu-helper'
import resolveRoot from '../resolve-root'
import hotPath from '../hot-path'
import flags from '../shared/util/feature-flags'

export default function () {
  const mainWindow = new Window(
    resolveRoot(`renderer/index.html?src=${hotPath('index.bundle.js')}`), {
      width: 1600,
      height: 1200,
      show: flags.login
    }
  )

  ipcMain.on('showMain', () => {
    mainWindow.show(true)
  })

  return mainWindow
}
