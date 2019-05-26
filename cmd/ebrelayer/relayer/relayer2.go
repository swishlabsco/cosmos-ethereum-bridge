// Package listener contains all the components for the events listener, which
// streams a list of future events.
package relayer

// import (
//   "sync"

//   log "github.com/golang/glog"
//   "github.com/pkg/errors"

//   "github.com/ethereum/go-ethereum/accounts/abi/bind"
//   "github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/model"
// )

// const (
//   eventRecvChanBufferSize = 1
// )

// // NewEventListener creates a new EventListener given the address of the contract to listen to.
// func NewEventListener(client bind.ContractBackend, watchers []model.ContractWatchers) *EventListener {
//   listener := &EventListener{
//     EventRecvChan: make(chan *model.Event, eventRecvChanBufferSize),
//     client:        client,
//     watchers:      watchers,
//     active:        false,
//   }
//   return listener
// }

// // EventListener handles the listener stream for events
// type EventListener struct {

//   // client is a ethereum backend from go-ethereum
//   client bind.ContractBackend

//   // EventRecvChan is the channel to send and receive Events
//   EventRecvChan chan *model.Event

//   // ActiveSubs is the list of active event subscriptions
//   ActiveSubs []utils.WatcherSubscription

//   Errors chan error

//   watchers []model.ContractWatchers

//   active bool

//   mutex sync.Mutex
// }

// // Start starts up the event listener and watchers
// func (l *EventListener) Start() error {
//   defer l.mutex.Unlock()
//   l.mutex.Lock()
//   l.Errors = make(chan error)
//   allSubs := []utils.WatcherSubscription{}
//   hasSubs := false
//   for _, watchers := range l.watchers {
//     newSubs, err := watchers.StartWatchers(
//       l.client,
//       l.EventRecvChan,
//       l.Errors,
//     )
//     if err != nil {
//       log.Errorf("Error starting watchers for %v at %v: err: %v",
//         watchers.ContractName(), watchers.ContractAddress(), err)
//     }
//     if len(newSubs) > 0 {
//       hasSubs = true
//     }
//     allSubs = append(allSubs, newSubs...)
//   }

//   if !hasSubs {
//     return errors.New("no watchers have been started")
//   }

//   l.ActiveSubs = allSubs
//   l.active = true
//   return nil
// }

// // AddWatchers will add watchersto the listener. If the listener is already
// // started, add to the list of watchers, start up with the watcher, and add it
// // to the list of subscriptions in the listener.
// // If the listener is not already started, will just be added to the list of watchers.
// func (l *EventListener) AddWatchers(w model.ContractWatchers) error {
//   defer l.mutex.Unlock()
//   l.mutex.Lock()
//   l.watchers = append(l.watchers, w)
//   if l.active {
//     _, err := w.StartWatchers(
//       l.client,
//       l.EventRecvChan,
//       l.Errors,
//     )
//     if err != nil {
//       log.Errorf("Error starting watchers for %v at %v: err: %v",
//         w.ContractName(), w.ContractAddress(), err)
//       return err
//     }
//   }
//   return nil
// }

// // RemoveWatchers will remove given watcher from the listener. If the listener is already
// // started, stop the watcher, removes the subscription, and removes from watcher list.
// // If the listener is not already started, will just be removed from the list of watchers.
// func (l *EventListener) RemoveWatchers(w model.ContractWatchers) error {
//   defer l.mutex.Unlock()
//   l.mutex.Lock()
//   if l.watchers != nil && len(l.watchers) > 0 {
//     for index, ew := range l.watchers {
//       if w.ContractAddress() == ew.ContractAddress() &&
//         w.ContractName() == ew.ContractName() {
//         if l.active {
//           _ = ew.StopWatchers(true) // nolint: gosec
//         }
//         // Delete the item in the watchers list.
//         copy(l.watchers[index:], l.watchers[index+1:])
//         l.watchers[len(l.watchers)-1] = nil
//         l.watchers = l.watchers[:len(l.watchers)-1]
//         return nil
//       }
//     }
//   }
//   return nil
// }

// // Stop shuts down the event listener and performs clean up
// func (l *EventListener) Stop(unsub bool) error {
//   defer l.mutex.Unlock()
//   l.mutex.Lock()
//   if l.watchers != nil && len(l.watchers) > 0 {
//     for _, w := range l.watchers {
//       _ = w.StopWatchers(unsub) // nolint: gosec
//     }
//   }
//   l.active = false
//   return nil
// }
