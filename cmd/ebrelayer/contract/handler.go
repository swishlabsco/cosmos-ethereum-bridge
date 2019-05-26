package contract

// import (
//   log "github.com/golang/glog"

//   "github.com/ethereum/go-ethereum/common"
// )

// func ContractWatchers(nameToAddrs map[string][]common.Address) []model.ContractWatchers {
//   watchers := []model.ContractWatchers{}

//   var addrs []common.Address
//   var addr common.Address
//   var ok bool

//   addrs, ok = nameToAddrs["peggy"]
//   if ok {
//     for _, addr = range addrs {
//       watch := watcher.NewNewsroomContractWatchers(addr)
//       watchers = append(watchers, watch)
//       log.Info("Added PeggyContract watcher")
//     }
//   }

//   return watchers
// }
