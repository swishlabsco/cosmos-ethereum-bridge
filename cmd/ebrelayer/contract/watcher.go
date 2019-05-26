package contract

// import (
//   "context"
//   "time"

//   "github.com/davecgh/go-spew/spew"
//   log "github.com/golang/glog"
//   "github.com/pkg/errors"

//   "github.com/ethereum/go-ethereum/accounts/abi/bind"
//   "github.com/ethereum/go-ethereum/common"
// )

// func NewPeggyContractWatchers(contractAddress common.Address) *PeggyContractWatchers {
//   return &PeggyContractWatchers{
//     contractAddress: contractAddress,
//   }
// }

// type PeggyContractWatchers struct {
//   errors          chan error
//   contractAddress common.Address
//   contract        *contract.PeggyContract
//   activeSubs      []utils.WatcherSubscription
// }

// func (w *PeggyContractWatchers) ContractAddress() common.Address {
//   return w.contractAddress
// }

// func (w *PeggyContractWatchers) ContractName() string {
//   return "PeggyContract"
// }

// func (w *PeggyContractWatchers) cancelFunc(cancelFn context.CancelFunc, killCancel <-chan bool) {
// }

// func (w *PeggyContractWatchers) StopWatchers(unsub bool) error {
//   if unsub {
//     for _, sub := range w.activeSubs {
//       sub.Unsubscribe()
//     }
//   }
//   w.activeSubs = nil
//   return nil
// }

// func (w *PeggyContractWatchers) StartWatchers(client bind.ContractBackend,
//   eventRecvChan chan *model.Event, errs chan error) ([]utils.WatcherSubscription, error) {
//   return w.StartPeggyContractWatchers(client, eventRecvChan, errs)
// }

// // StartPeggyContractWatchers starts up the event watchers for PeggyContract
// func (w *PeggyContractWatchers) StartPeggyContractWatchers(client bind.ContractBackend,
//   eventRecvChan chan *model.Event, errs chan error) ([]utils.WatcherSubscription, error) {
//   w.errors = errs
//   contract, err := contract.NewPeggyContract(w.contractAddress, client)
//   if err != nil {
//     log.Errorf("Error initializing StartPeggyContract: err: %v", err)
//     return nil, errors.Wrap(err, "error initializing StartPeggyContract")
//   }
//   w.contract = contract

//   var sub utils.WatcherSubscription
//   subs := []utils.WatcherSubscription{}

//   sub, err = w.startWatchLogLock(eventRecvChan)
//   if err != nil {
//     return nil, errors.WithMessage(err, "error starting startLogLock")
//   }
//   subs = append(subs, sub)

//   w.activeSubs = subs
//   return subs, nil
// }

// func (w *PeggyContractWatchers) startWatchLogLock(eventRecvChan chan *model.Event) (utils.WatcherSubscription, error) {
//   killCancelTimeoutSecs := 10
//   preemptiveTimeoutSecs := 60 * 30
//   return utils.NewWatcherSubscription("WatchLogLock", func(quit <-chan struct{}) error {
//     startupFn := func() (utils.WatcherSubscription, chan *contract.PeggyContractLogLock, error) {
//       ctx := context.Background()
//       ctx, cancelFn := context.WithCancel(ctx)
//       opts := &bind.WatchOpts{Context: ctx}
//       killCancel := make(chan bool)
//       // 10 sec timeout mechanism for starting up watcher
//       go func(cancelFn context.CancelFunc, killCancel <-chan bool) {
//         select {
//         case <-time.After(time.Duration(killCancelTimeoutSecs) * time.Second):
//           log.Errorf("WatchLogLock start timeout, cancelling...")
//           cancelFn()
//         case <-killCancel:
//         }
//       }(cancelFn, killCancel)
//       recvChan := make(chan *contract.PeggyContractLogLock)
//       log.Infof("startupFn: Starting WatchLogLock")
//       sub, err := w.contract.WatchLogLock(
//         opts,
//         recvChan,
//       )
//       close(killCancel)
//       if err != nil {
//         if sub != nil {
//           log.Infof("startupFn: Unsubscribing WatchLogLock")
//           sub.Unsubscribe()
//         }
//         return nil, nil, errors.Wrap(err, "startupFn: error starting WatchLogLock")
//       }
//       log.Infof("startupFn: WatchLogLock started")
//       return sub, recvChan, nil
//     }
//     sub, recvChan, err := startupFn()
//     if err != nil {
//       log.Errorf("Error starting WatchLogLock: %v", err)
//       if sub != nil {
//         sub.Unsubscribe()
//       }
//       w.errors <- err
//       return err
//     }
//     defer sub.Unsubscribe()
//     log.Infof("Starting up WatchLogLock for contract %v", w.contractAddress.Hex())
//     for {
//       select {
//       // 30 min premptive resubscribe
//       case <-time.After(time.Second * time.Duration(preemptiveTimeoutSecs)):
//         log.Infof("Premptive restart of LogLock")
//         oldSub := sub
//         sub, recvChan, err = startupFn()
//         if err != nil {
//           log.Errorf("Error starting LogLock: %v", err)
//           w.errors <- err
//           return err
//         }
//         log.Infof("Attempting to unsub old LogLock")
//         oldSub.Unsubscribe()
//         log.Infof("Done preemptive restart LogLock")
//       case event := <-recvChan:
//         if log.V(2) {
//           log.Infof("Received event on WatchLogLock: %v", spew.Sprintf("%#+v", event))
//         } else {
//           log.Info("Received event on WatchLogLock")
//         }
//         modelEvent, err := model.NewEventFromContractEvent("LogLock", w.ContractName(), w.contractAddress, event, ctime.CurrentEpochSecsInInt64(), model.Watcher)
//         if err != nil {
//           log.Errorf("Error creating new event: event: %v, err: %v", event, err)
//           continue
//         }
//         select {
//         case eventRecvChan <- modelEvent:
//           if log.V(2) {
//             log.Infof("Sent event to eventRecvChan on WatchLogLock: %v", spew.Sprintf("%#+v", event))
//           } else {
//             log.Info("Sent event to eventRecvChan on WatchLogLock")
//           }
//         case err := <-sub.Err():
//           log.Errorf("Error with WatchLogLock, fatal (a): %v", err)
//           err = errors.Wrap(err, "error with WatchLogLock")
//           w.errors <- err
//           return err
//         case <-quit:
//           log.Infof("Quit WatchLogLock (a): %v", err)
//           return nil
//         }
//       case err := <-sub.Err():
//         log.Errorf("Error with WatchLogLock, fatal (b): %v", err)
//         err = errors.Wrap(err, "error with WatchLogLock")
//         w.errors <- err
//         return err
//       case <-quit:
//         log.Infof("Quitting loop for WatchLogLock")
//         return nil
//       }
//     }
//   }), nil
// }