# K8sTA Roadmap

> 🟡&nbsp;&nbsp;K8sTA is highly experimental at this time and breaking changes
> should be anticipated between pre-GA minor releases.

The current goal with K8sTA is to discover and add value in small increments. If
the road ahead were more clear, we'd document it here -- and someday we will. In
the meantime, we'll document the road _behind us_ and _currently under our
tires_ to highlight what we've already experimented with, or are experimenting
with currently, what has worked, and what hasn't.

## Iteration 1

In the first iteration, our goal was only to move a new Docker image through a
series of environments _with no manual intervention and no "glue code"
required._

We introduced a CRD, `Track`, instances of which subscribe to new images pushed
to different image repositories (in Docker Hub only for now). Instances of a
`Track` also define an ordered list of environments to progress new images
through. In this initial iteration, an environment is just a reference to an
existing Argo CD `Application`.

When an inbound webhook from Docker Hub is received by the server component and
indicates a new image is to be progressed along a subscribed `Track`, an
instance of a `Ticket` CRD is created. A `TicketReconciler` in the controller
component manages progressive deployment of the image along the `Track`.

Argo CD operates 100% as normal and K8sTA is purely complementary.

The results of the first iteration were successfully demoed on 2022-08-03 and
a recording of that demo is available to Akuity employees
[here](https://drive.google.com/file/d/1HfAaS9tky3QVof9xTvYugr55CwIhCOSJ/view?usp=sharing).
