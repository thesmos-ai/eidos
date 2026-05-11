// Copyright Thesmos B.V. 2026
// SPDX-License-Identifier: MIT

package meta

// Observer is the callback fired by [Bag.AddObserver] for each Set
// operation against the bag. The supplied name is the metadata key
// that was just set; the observer typically uses the name plus the
// owning bag's identity (captured by the registering closure) to
// update an external index.
//
// Observers fire synchronously inside the mutating call, holding
// the bag's write lock. The implementation must NOT call back into
// the same bag from inside the callback or deadlock results; the
// callback's job is to record the event and return promptly.
//
// Observers are not fired for tombstones or prefix tombstones — the
// store-side by-metadata-key index treats Set as the indexed event
// and re-resolves [Bag.Has] at query time to handle tombstones.
type Observer func(name string)

// AddObserver registers fn to receive notifications for future Set
// operations on the bag. Observers fire in registration order.
// AddObserver itself is safe for concurrent use.
func (b *Bag) AddObserver(fn Observer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.observers = append(b.observers, fn)
}

// fireObservers calls each registered observer with name. The caller
// must already hold the bag's write lock so the fire happens-before
// the lock release.
func (b *Bag) fireObservers(name string) {
	for _, fn := range b.observers {
		fn(name)
	}
}
