package core

import (
	"github.com/stojg/vector"
	"sync"
)

func NewBody(invMass float64) *Body {
	body := &Body{
		velocity:                  &vector.Vector3{},
		rotation:                  &vector.Vector3{},
		Forces:                    &vector.Vector3{},
		transformMatrix:           &vector.Matrix4{},
		InverseInertiaTensor:      &vector.Matrix3{},
		InverseInertiaTensorWorld: &vector.Matrix3{},
		ForceAccum:                &vector.Vector3{},
		TorqueAccum:               &vector.Vector3{},
		maxAcceleration:           &vector.Vector3{100, 100, 100},
		Acceleration:              &vector.Vector3{},
		LinearDamping:             0.99,
		AngularDamping:            0.99,
		maxRotation:               3.14 / 1,
		InvMass:                   invMass,
		CanSleep:                  true,
		isAwake:                   true,
		SleepEpsilon:              0.00001,
	}

	it := &vector.Matrix3{}
	it.SetBlockInertiaTensor(&vector.Vector3{1, 1, 1}, 1/invMass)
	body.SetInertiaTensor(it)
	return body
}

type Body struct {
	Component
	sync.Mutex
	// Holds the linear velocity of the rigid body in world space.
	velocity *vector.Vector3
	// Holds the angular velocity, or rotation for the rigid body in world space.
	rotation *vector.Vector3

	// Holds the inverse of the mass of the rigid body. It is more useful to hold the inverse mass
	// because integration is simpler, and because in real time simulation it is more useful to have
	// bodies with infinite mass (immovable) than zero mass (completely unstable in numerical
	// simulation).
	InvMass float64
	// Holds the inverse of the body's inertia tensor. The inertia tensor provided must not be
	// degenerate (that would mean the body had zero inertia for spinning along one axis). As long
	// as the tensor is finite, it will be invertible. The inverse tensor is used for similar
	// reasons to the use of inverse mass.
	// The inertia tensor, unlike the other variables that define a rigid body, is given in body
	// space.
	InverseInertiaTensor *vector.Matrix3
	// Holds the amount of damping applied to linear motion.  Damping is required to remove energy
	// added through numerical instability in the integrator.
	LinearDamping float64
	// Holds the amount of damping applied to angular motion.  Damping is required to remove energy
	// added through numerical instability in the integrator.
	AngularDamping float64

	/**
	 * Derived Data
	 *
	 * These data members hold information that is derived from the other data in the class.
	 */

	// Holds the inverse inertia tensor of the body in world space. The inverse inertia tensor
	// member is specified in the body's local space. @see inverseInertiaTensor
	InverseInertiaTensorWorld *vector.Matrix3
	// Holds the amount of motion of the body. This is a recency weighted mean that can be used to
	// put a body to sleap.
	Motion float64
	// A body can be put to sleep to avoid it being updated by the integration functions or affected
	// by collisions with the world.
	isAwake bool
	// Some bodies may never be allowed to fall asleep. User controlled bodies, for example, should
	// be always awake.
	CanSleep bool
	// Holds a transform matrix for converting body space into world space and vice versa. This can
	// be achieved by calling the getPointIn*Space functions.
	transformMatrix *vector.Matrix4

	/**
	 * Force and Torque Accumulators
	 *
	 * These data members store the current force, torque and acceleration of the rigid body. Forces
	 * can be added to the rigid body in any order, and the class decomposes them into their
	 * constituents, accumulating them for the next simulation step. At the simulation step, the
	 * accelerations are calculated and stored to be applied to the rigid body.
	 */

	// Holds the accumulated force to be applied at the next integration step.
	ForceAccum *vector.Vector3

	// Holds the accumulated torque to be applied at the next integration step.
	TorqueAccum *vector.Vector3

	// Holds the acceleration of the rigid body.  This value can be used to set acceleration due to
	// gravity (its primary use), or any other constant acceleration.
	Acceleration *vector.Vector3

	maxAcceleration *vector.Vector3

	// limits the linear acceleration
	MaxAngularAcceleration *vector.Vector3
	// limits the angular velocity
	maxRotation float64

	// Holds the linear acceleration of the rigid body, for the previous frame.
	LastFrameAcceleration *vector.Vector3

	SleepEpsilon float64
	Forces       *vector.Vector3
}

func (g *Body) Position() *vector.Vector3 {
	return g.transform.position
}

func (g *Body) Rotation() *vector.Vector3 {
	return g.rotation
}

func (g *Body) MaxRotation() float64 {
	return g.maxRotation
}

func (g *Body) Orientation() *vector.Quaternion {
	return g.transform.orientation
}

func (g *Body) MaxAcceleration() *vector.Vector3 {
	return g.maxAcceleration
}

func (g *Body) Velocity() *vector.Vector3 {
	return g.velocity
}

func (rb *Body) Mass() float64 {
	return 1 / rb.InvMass
}

func (rb *Body) SetInertiaTensor(inertiaTensor *vector.Matrix3) {
	rb.InverseInertiaTensor.SetInverse(inertiaTensor)
}

func (rb *Body) AddForce(force *vector.Vector3) {
	rb.ForceAccum.Add(force)
	rb.SetAwake(true)
}

func (rb *Body) AddForceAtBodyPoint(ent *Transform, force, point *vector.Vector3) {
	// convert to coordinates relative to center of mass
	pt := rb.GetPointInWorldSpace(point)
	rb.AddForceAtPoint(ent, force, pt)
	rb.SetAwake(true)
}

func (rb *Body) AddForceAtPoint(body *Transform, force, point *vector.Vector3) {
	// convert to coordinates relative to center of mass
	pt := point.NewSub(body.position)
	rb.ForceAccum.Add(force)
	rb.TorqueAccum.Add(pt.NewCross(force))
	rb.SetAwake(true)
}

func (rb *Body) AddTorque(torque *vector.Vector3) {
	rb.TorqueAccum.Add(torque)
	rb.SetAwake(true)
}

func (rb *Body) ClearAccumulators() {
	rb.Forces.Clear()
	rb.ForceAccum.Clear()
	rb.TorqueAccum.Clear()
}

func (rb *Body) GetPointInWorldSpace(point *vector.Vector3) *vector.Vector3 {
	return rb.transformMatrix.TransformVector3(point)
}

func (rb *Body) getTransform() *vector.Matrix4 {
	return rb.transformMatrix
}

func (rb *Body) CalculateDerivedData(transform *Transform) {
	transform.Orientation().Normalize()
	rb.calculateTransformMatrix(rb.transformMatrix, transform.position, transform.Orientation())
	rb.transformInertiaTensor(rb.InverseInertiaTensorWorld, transform.Orientation(), rb.InverseInertiaTensor, rb.transformMatrix)
}

func (rb *Body) Awake() bool {
	rb.Lock()
	defer rb.Unlock()
	return rb.isAwake
}

func (rb *Body) SetAwake(t bool) {
	rb.Lock()
	defer rb.Unlock()
	rb.isAwake = t
}

/**
 * Internal function to do an intertia tensor transform by a vector.Quaternion.
 * Note that the implementation of this function was created by an
 * automated code-generator and optimizer.
 */
func (rb *Body) transformInertiaTensor(iitWorld *vector.Matrix3, q *vector.Quaternion, iitBody *vector.Matrix3, rotmat *vector.Matrix4) {
	t4 := rotmat[0]*iitBody[0] + rotmat[1]*iitBody[3] + rotmat[2]*iitBody[6]
	t9 := rotmat[0]*iitBody[1] + rotmat[1]*iitBody[4] + rotmat[2]*iitBody[7]
	t14 := rotmat[0]*iitBody[2] + rotmat[1]*iitBody[5] + rotmat[2]*iitBody[8]
	t28 := rotmat[4]*iitBody[0] + rotmat[5]*iitBody[3] + rotmat[6]*iitBody[6]
	t33 := rotmat[4]*iitBody[1] + rotmat[5]*iitBody[4] + rotmat[6]*iitBody[7]
	t38 := rotmat[4]*iitBody[2] + rotmat[5]*iitBody[5] + rotmat[6]*iitBody[8]
	t52 := rotmat[8]*iitBody[0] + rotmat[9]*iitBody[3] + rotmat[10]*iitBody[6]
	t57 := rotmat[8]*iitBody[1] + rotmat[9]*iitBody[4] + rotmat[10]*iitBody[7]
	t62 := rotmat[8]*iitBody[2] + rotmat[9]*iitBody[5] + rotmat[10]*iitBody[8]

	iitWorld[0] = t4*rotmat[0] + t9*rotmat[1] + t14*rotmat[2]
	iitWorld[1] = t4*rotmat[4] + t9*rotmat[5] + t14*rotmat[6]
	iitWorld[2] = t4*rotmat[8] + t9*rotmat[9] + t14*rotmat[10]
	iitWorld[3] = t28*rotmat[0] + t33*rotmat[1] + t38*rotmat[2]
	iitWorld[4] = t28*rotmat[4] + t33*rotmat[5] + t38*rotmat[6]
	iitWorld[5] = t28*rotmat[8] + t33*rotmat[9] + t38*rotmat[10]
	iitWorld[6] = t52*rotmat[0] + t57*rotmat[1] + t62*rotmat[2]
	iitWorld[7] = t52*rotmat[4] + t57*rotmat[5] + t62*rotmat[6]
	iitWorld[8] = t52*rotmat[8] + t57*rotmat[9] + t62*rotmat[10]
}

/**
 * Inline function that creates a transform matrix from a
 * position and orientation.
 */
func (rb *Body) calculateTransformMatrix(transformMatrix *vector.Matrix4, position *vector.Vector3, orientation *vector.Quaternion) {

	transformMatrix[0] = 1 - 2*orientation.J*orientation.J - 2*orientation.K*orientation.K
	transformMatrix[1] = 2*orientation.I*orientation.J - 2*orientation.R*orientation.K
	transformMatrix[2] = 2*orientation.I*orientation.K + 2*orientation.R*orientation.J
	transformMatrix[3] = position[0]

	transformMatrix[4] = 2*orientation.I*orientation.J + 2*orientation.R*orientation.K
	transformMatrix[5] = 1 - 2*orientation.I*orientation.I - 2*orientation.K*orientation.K
	transformMatrix[6] = 2*orientation.J*orientation.K - 2*orientation.R*orientation.I
	transformMatrix[7] = position[1]

	transformMatrix[8] = 2*orientation.I*orientation.K - 2*orientation.R*orientation.J
	transformMatrix[9] = 2*orientation.J*orientation.K + 2*orientation.R*orientation.I
	transformMatrix[10] = 1 - 2*orientation.I*orientation.I - 2*orientation.J*orientation.J
	transformMatrix[11] = position[2]
}
