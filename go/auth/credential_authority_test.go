package auth

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	libkb "github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/logger"
	keybase1 "github.com/keybase/client/go/protocol"
	context "golang.org/x/net/context"
	"sync"
	"testing"
	"time"
)

type testUser struct {
	uid      keybase1.UID
	username libkb.NormalizedUsername
	sibkeys  []keybase1.KID
	subkeys  []keybase1.KID
}

type testState struct {
	sync.Mutex

	users     map[keybase1.UID](*testUser)
	changes   []keybase1.UID
	now       time.Time
	evictCh   chan keybase1.UID
	pokeCh    chan struct{}
	startOnce sync.Once
	numGets   int
}

var seq uint32

func genKID() keybase1.KID {
	var kid [35]byte
	kid[0] = 0x1
	kid[1] = 0x20
	binary.BigEndian.PutUint32(kid[30:34], seq)
	seq++
	kid[34] = 0xa0
	return keybase1.KIDFromSlice(kid[:])
}

func genUsername() string {
	w, _ := libkb.SecWordList(1)
	var buf [4]byte
	rand.Read(buf[:])
	return fmt.Sprintf("%s%x", w[0], buf)
}

func newTestUser(nKeys int) *testUser {
	un := genUsername()
	ret := testUser{
		username: libkb.NewNormalizedUsername(un),
		uid:      libkb.UsernameToUID(un),
		sibkeys:  make([]keybase1.KID, nKeys),
		subkeys:  make([]keybase1.KID, nKeys),
	}
	for i := 0; i < nKeys; i++ {
		ret.sibkeys[i] = genKID()
		ret.subkeys[i] = genKID()
	}
	return &ret
}

func (ts *testState) newTestUser(nKeys int) *testUser {
	ts.Lock()
	defer ts.Unlock()
	ret := newTestUser(nKeys)
	ts.users[ret.uid] = ret
	return ret
}

func (ts *testState) mutateUser(uid keybase1.UID, mutator func(u *testUser)) bool {
	ts.Lock()
	defer ts.Unlock()
	u := ts.users[uid]
	if u == nil {
		return false
	}
	mutator(u)
	ts.changes = append(ts.changes, uid)
	return true
}

func newTestState() *testState {
	return &testState{
		users:   make(map[keybase1.UID](*testUser)),
		now:     time.Unix(100, 0),
		evictCh: make(chan keybase1.UID, 1),
		pokeCh:  make(chan struct{}),
	}
}

type userNotFoundError struct {
}

func (e userNotFoundError) Error() string {
	return "user not found"
}

func (ts *testState) GetUser(_ context.Context, uid keybase1.UID) (
	un libkb.NormalizedUsername, sibkeys, subkeys []keybase1.KID, err error) {
	ts.Lock()
	defer ts.Unlock()
	u := ts.users[uid]
	if u == nil {
		return libkb.NormalizedUsername(""), nil, nil, userNotFoundError{}
	}
	ts.numGets++
	return u.username, u.sibkeys, u.subkeys, nil
}

func (ts *testState) PollForChanges(_ context.Context) ([]keybase1.UID, error) {
	ts.Lock()
	defer ts.Unlock()
	ret := ts.changes
	ts.changes = nil
	return ret, nil
}

var _ UserKeyAPIer = (*testState)(nil)
var _ engine = (*testState)(nil)

func (ts *testState) tick(d time.Duration) {
	ts.pokeCh <- struct{}{}
	ts.Lock()
	ts.now = ts.now.Add(d)
	ts.Unlock()
	ts.pokeCh <- struct{}{}
}

func (ts *testState) Now() time.Time {
	ts.Lock()
	ret := ts.now
	ts.Unlock()
	return ret
}

func (ts *testState) GetPokeCh() <-chan struct{} { return ts.pokeCh }

func (ts *testState) Evicted(uid keybase1.UID) {
	ts.evictCh <- uid
}

func newTestSetup() (*testState, *CredentialAuthority) {
	s := newTestState()
	c := newCredentialAuthorityWithEngine(logger.New("test", libkb.ErrorWriter()), s, s)
	return s, c
}

func TestSimple(t *testing.T) {
	S, C := newTestSetup()
	u0 := S.newTestUser(4)

	key0 := u0.sibkeys[0]
	key1 := u0.sibkeys[1]

	if S.numGets != 0 {
		t.Fatal("expected 0 gets")
	}

	err := C.CheckUserKey(context.TODO(), u0.uid, &u0.username, &key0)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 1 {
		t.Fatal("expected 1 get")
	}
	err = C.CheckUserKey(context.TODO(), u0.uid, &u0.username, &key0)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 1 {
		t.Fatal("expected 1 get")
	}

	S.mutateUser(u0.uid, func(u *testUser) {
		u.sibkeys = u.sibkeys[1:]
	})

	// Advance just an iota, so that our polling of the server
	// has a chance to complete.
	S.tick(pollWait)

	// wait for the first eviction
	uid := <-S.evictCh
	if uid != u0.uid {
		t.Fatalf("Wrong UID on eviction: %s != %s\n", uid, u0.uid)
	}

	err = C.CheckUserKey(context.TODO(), u0.uid, &u0.username, &key0)
	if err == nil {
		t.Fatal("Expected an error")
	} else if bke, ok := err.(BadKeyError); !ok {
		t.Fatal("Expected a bad key error")
	} else if bke.uid != u0.uid {
		t.Fatalf("Expected a bad key error on %s (not %s)", u0.uid, bke.uid)
	} else if bke.kid != key0 {
		t.Fatalf("Expected a bad key error on key %s (not %s)", key0, bke.kid)
	}

	err = C.CheckUserKey(context.TODO(), u0.uid, &u0.username, &key1)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 2 {
		t.Fatal("expected 2 gets")
	}
	S.tick(userTimeout + time.Millisecond)
	err = C.CheckUserKey(context.TODO(), u0.uid, &u0.username, &key1)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 3 {
		t.Fatal("expected 3 gets")
	}
	S.tick(cacheTimeout + time.Millisecond)

	// u0 should now be gone since we haven't touched him in over cacheTimeout
	// duration.
	uid = <-S.evictCh
	if uid != u0.uid {
		t.Fatalf("Wrong UID on eviction: %s != %s\n", uid, u0.uid)
	}

	// Make a new user -- u1!
	u1 := S.newTestUser(4)

	ng := 3
	for i := 0; i < 10; i++ {
		err = C.CheckUserKey(context.TODO(), u1.uid, &u1.username, &u1.sibkeys[0])
		if err != nil {
			t.Fatal(err)
		}
		ng++
		if S.numGets != ng {
			t.Fatalf("expected %d gets, got %d", ng, S.numGets)
		}
		S.tick(userTimeout + time.Millisecond)

		select {
		case uid = <-S.evictCh:
			t.Fatalf("Got unwanted eviction for %s", uid)
		default:
		}
	}

	S.tick(cacheTimeout - userTimeout + 3*time.Millisecond)
	uid = <-S.evictCh
	if uid != u1.uid {
		t.Fatalf("Got wrong eviction: wanted %s but got %s\n", u1.uid, uid)
	}

	// Make a new user -- u2!
	u2 := S.newTestUser(4)
	err = C.CheckUserKey(context.TODO(), u2.uid, &u2.username, &u2.sibkeys[0])
	if err != nil {
		t.Fatal(err)
	}
	ng++
	if S.numGets != ng {
		t.Fatalf("expected %d gets, got %d", ng, S.numGets)
	}

	// Check that u2 is evicted properly after we shutdown the CA.
	C.Shutdown()
	uid = <-S.evictCh
	if uid != u2.uid {
		t.Fatalf("Got wrong eviction: wanted %s but got %s\n", u2.uid, uid)
	}

}

func TestCheckUsers(t *testing.T) {
	S, C := newTestSetup()

	var users, usersWithDud []keybase1.UID
	for i := 0; i < 10; i++ {
		u := S.newTestUser(2)
		users = append(users, u.uid)
		usersWithDud = append(usersWithDud, u.uid)
	}
	usersWithDud = append(usersWithDud, libkb.UsernameToUID(genUsername()))

	if S.numGets != 0 {
		t.Fatal("expected 0 gets")
	}

	err := C.CheckUsers(context.TODO(), users)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 10 {
		t.Fatal("expected 10 gets")
	}
	err = C.CheckUsers(context.TODO(), users)
	if err != nil {
		t.Fatal(err)
	}
	if S.numGets != 10 {
		t.Fatal("expected 10 gets")
	}

	err = C.CheckUsers(context.TODO(), usersWithDud)
	if err == nil {
		t.Fatal("Expected an error")
	} else if _, ok := err.(userNotFoundError); !ok {
		t.Fatal("Expected a user not found error")
	}
	C.Shutdown()
}

func TestCompareKeys(t *testing.T) {
	S, C := newTestSetup()
	u := S.newTestUser(10)

	err := C.CompareUserKeys(context.TODO(), u.uid, u.sibkeys, u.subkeys)
	if err != nil {
		t.Fatal(err)
	}

	err = C.CompareUserKeys(context.TODO(), u.uid, nil, u.subkeys)
	if err != nil {
		t.Fatal(err)
	}

	err = C.CompareUserKeys(context.TODO(), u.uid, u.sibkeys, nil)
	if err != nil {
		t.Fatal(err)
	}

	missingSibkey := u.sibkeys[1:]
	err = C.CompareUserKeys(context.TODO(), u.uid, missingSibkey, u.subkeys)
	if err == nil {
		t.Fatal("Expected an error")
	} else if _, ok := err.(KeysNotEqualError); !ok {
		t.Fatal("Expected keys not equal error")
	}

	missingSubkey := u.subkeys[1:]
	err = C.CompareUserKeys(context.TODO(), u.uid, u.sibkeys, missingSubkey)
	if err == nil {
		t.Fatal("Expected an error")
	} else if _, ok := err.(KeysNotEqualError); !ok {
		t.Fatal("Expected keys not equal error")
	}
	C.Shutdown()
}
