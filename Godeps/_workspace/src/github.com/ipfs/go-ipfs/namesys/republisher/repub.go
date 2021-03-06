package republisher

import (
	"errors"
	"sync"
	"time"

	key "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/blocks/key"
	namesys "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/namesys"
	pb "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/namesys/pb"
	peer "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/p2p/peer"
	path "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/path"
	"github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/routing"
	dhtpb "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/routing/dht/pb"

	proto "github.com/noffle/ipget/Godeps/_workspace/src/github.com/gogo/protobuf/proto"
	logging "github.com/noffle/ipget/Godeps/_workspace/src/github.com/ipfs/go-ipfs/vendor/go-log-v1.0.0"
	ds "github.com/noffle/ipget/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	goprocess "github.com/noffle/ipget/Godeps/_workspace/src/github.com/jbenet/goprocess"
	gpctx "github.com/noffle/ipget/Godeps/_workspace/src/github.com/jbenet/goprocess/context"
	context "github.com/noffle/ipget/Godeps/_workspace/src/golang.org/x/net/context"
)

var errNoEntry = errors.New("no previous entry")

var log = logging.Logger("ipns-repub")

var DefaultRebroadcastInterval = time.Hour * 4

const DefaultRecordLifetime = time.Hour * 24

type Republisher struct {
	r  routing.IpfsRouting
	ds ds.Datastore
	ps peer.Peerstore

	Interval time.Duration

	// how long records that are republished should be valid for
	RecordLifetime time.Duration

	entrylock sync.Mutex
	entries   map[peer.ID]struct{}
}

func NewRepublisher(r routing.IpfsRouting, ds ds.Datastore, ps peer.Peerstore) *Republisher {
	return &Republisher{
		r:              r,
		ps:             ps,
		ds:             ds,
		entries:        make(map[peer.ID]struct{}),
		Interval:       DefaultRebroadcastInterval,
		RecordLifetime: DefaultRecordLifetime,
	}
}

func (rp *Republisher) AddName(id peer.ID) {
	rp.entrylock.Lock()
	defer rp.entrylock.Unlock()
	rp.entries[id] = struct{}{}
}

func (rp *Republisher) Run(proc goprocess.Process) {
	tick := time.NewTicker(rp.Interval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			err := rp.republishEntries(proc)
			if err != nil {
				log.Error("Republisher failed to republish: ", err)
			}
		case <-proc.Closing():
			return
		}
	}
}

func (rp *Republisher) republishEntries(p goprocess.Process) error {
	ctx, cancel := context.WithCancel(gpctx.OnClosingContext(p))
	defer cancel()

	for id, _ := range rp.entries {
		log.Debugf("republishing ipns entry for %s", id)
		priv := rp.ps.PrivKey(id)

		// Look for it locally only
		_, ipnskey := namesys.IpnsKeysForID(id)
		p, seq, err := rp.getLastVal(ipnskey)
		if err != nil {
			if err == errNoEntry {
				continue
			}
			return err
		}

		// update record with same sequence number
		eol := time.Now().Add(rp.RecordLifetime)
		err = namesys.PutRecordToRouting(ctx, priv, p, seq, eol, rp.r, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rp *Republisher) getLastVal(k key.Key) (path.Path, uint64, error) {
	ival, err := rp.ds.Get(k.DsKey())
	if err != nil {
		// not found means we dont have a previously published entry
		return "", 0, errNoEntry
	}

	val := ival.([]byte)
	dhtrec := new(dhtpb.Record)
	err = proto.Unmarshal(val, dhtrec)
	if err != nil {
		return "", 0, err
	}

	// extract published data from record
	e := new(pb.IpnsEntry)
	err = proto.Unmarshal(dhtrec.GetValue(), e)
	if err != nil {
		return "", 0, err
	}
	return path.Path(e.Value), e.GetSequence(), nil
}
