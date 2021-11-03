package app

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/infastin/t11go/internal/mount"
	"github.com/infastin/t11go/internal/view"
)

const appID = "com.github.infastin.t11go"

type Application struct {
	fyne.App
	win     fyne.Window
	watcher mount.Watcher
	view    *view.View
	mounts  []mount.Mount
}

func NewApplication() (*Application, error) {
	app := app.NewWithID(appID)
	win := app.NewWindow("T11Go")

	app.Settings().SetTheme(theme.LightTheme())

	watcher, err := mount.NewWatcher()
	if err != nil {
		return nil, err
	}

	mounts := watcher.Mounts()

	view := view.NewView()
	content := view.BuildUI(mounts)

	win.SetContent(content)
	win.Resize(fyne.NewSize(640, 480))

	return &Application{
		App:     app,
		win:     win,
		watcher: watcher,
		view:    view,
		mounts:  mounts,
	}, nil
}

func (app *Application) Watch() error {
	contains := func(mnts []mount.Mount, mount mount.Mount) int {
		for i, mnt := range mnts {
			if mnt.Device == mount.Device {
				return i
			}
		}

		return -1
	}

	err := app.watcher.Watch()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-app.watcher.Events():
				oldMounts := app.mounts
				newMounts := app.watcher.Mounts()

				for _, oldMnt := range oldMounts {
					if i := contains(newMounts, oldMnt); i != -1 {
						if oldMnt != newMounts[i] {
							app.view.UpdateTab(newMounts[i])
						}
					} else {
						app.view.RemoveTab(oldMnt.Device)
					}
				}

				for _, newMnt := range newMounts {
					if i := contains(oldMounts, newMnt); i == -1 {
						app.view.AddTab(newMnt)
					}
				}

				app.mounts = newMounts
			case err := <-app.watcher.Errors():
				fmt.Println(err)
				return
			}
		}
	}()

	return nil
}

func (app *Application) Run() {
	app.win.ShowAndRun()
}
