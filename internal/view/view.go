package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/infastin/t11go/internal/mount"
)

type item struct {
	next *item

	mntPoint  *widget.Label
	mntDevice *widget.Label
	mntFtype  *widget.Label
	mntSize   *widget.Label
	mntAvail  *widget.Label
}

func newItem(head *item, mnt mount.Mount) (*item, *item) {
	newItem := &item{
		mntPoint:  widget.NewLabel(mnt.Mpoint),
		mntDevice: widget.NewLabel(mnt.Device),
		mntFtype:  widget.NewLabel(mnt.Ftype),
		mntSize:   widget.NewLabel(mnt.Size()),
		mntAvail:  widget.NewLabel(mnt.Avail()),
	}

	if head != nil {
		it := head
		for it.next != nil {
			it = it.next
		}
		it.next = newItem

		return head, newItem
	}

	return newItem, newItem
}

type View struct {
	tabs  *container.AppTabs
	items *item
}

func NewView() *View {
	return &View{
		tabs: container.NewAppTabs(),
	}
}

func (v *View) AddTab(mnt mount.Mount) {
	head, it := newItem(v.items, mnt)
	v.items = head

	tab := container.NewPadded(widget.NewForm(
		widget.NewFormItem("Mount Point", it.mntPoint),
		widget.NewFormItem("Device", it.mntDevice),
		widget.NewFormItem("File System", it.mntFtype),
		widget.NewFormItem("Size", it.mntSize),
		widget.NewFormItem("Available", it.mntAvail),
	))

	tabItem := container.NewTabItem(mnt.Device, tab)
	v.tabs.Append(tabItem)
}

func (v *View) RemoveTab(device string) {
	tabItems := v.tabs.Items
	for _, item := range tabItems {
		if item.Text == device {
			v.tabs.Remove(item)
			return
		}
	}

	prev := &v.items
	it := v.items

	for it != nil {
		if it.mntDevice.Text == device {
			*prev = it.next
			return
		}

		prev = &it.next
		it = it.next
	}
}

func (v *View) UpdateTab(mount mount.Mount) {
	it := v.items

	for it != nil {
		if it.mntDevice.Text == mount.Device {
			it.mntPoint.SetText(mount.Mpoint)
			it.mntSize.SetText(mount.Size())
			it.mntAvail.SetText(mount.Avail())
			return
		}

		it = it.next
	}
}

func (v *View) BuildUI(mounts []mount.Mount) fyne.CanvasObject {
	for _, mnt := range mounts {
		v.AddTab(mnt)
	}

	return v.tabs
}
