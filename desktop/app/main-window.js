import Window from './window'
import {ipcMain} from 'electron'
import menuHelper from './menu-helper'
import resolveRoot from '../resolve-root'
import hotPath from '../hot-path'
import {allowLogin} from '../shared/util/feature-flags'

export default function () {
  const mainWindow = new Window(
    resolveRoot(`renderer/index.html?src=${hotPath('index.bundle.js')}`), {
      width: 1600,
      height: 1200,
      show: allowLogin
    }
  )

  ipcMain.on('showMain', () => {
    mainWindow.show(true)
    menuHelper(mainWindow.window)
  })

  return mainWindow
}
