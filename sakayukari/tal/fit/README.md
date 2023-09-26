two things:
- modeled velocity over time
- average velocity in an interval (multiple)

for each avg velocity:
  let vm = avg modeled velocity
  let va = actual avg velocity
  let delta = va-vm
  save the delta in this interval
  TODO: how does this work if the velocty changes

example:
  0.5*10m/s + 0.5*20m/s
  recorded avg is 17m/s
  delta = 2m/s
  (0.5*10m/s + 0.5*20m/s)+2
  0.5*12m/s + 0.5*22m/s
  hmm try this out ig

- record power changes in time
  - e.g. `[(0s, 0), (1s, 0→100), (2s, 100→0)]`
- measurement section = section between two measurements of train
- calculate the average speed between measurement sections using power changes
- to change f(power)=speed function
  - half of time was speed 10, other half was speed 20
  - speed by function=f(10*0.5+20*0.5)
  - given actual speed
  - d=delta between speed given by function and measured
  - add these new points to the function and re-polyfit (10, f(10)+0.5*d) (20, f(20)+0.5*d)
