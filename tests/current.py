from mumax2 import *

# Electrical current paths

Nx = 8
Ny = 8
Nz = 1
setgridsize(Nx, Ny, Nz)
setcellsize(500e-9/Nx, 125e-9/Ny, 3e-9/Nz)

load('current')

save('kern_el', 'gplot', [], 'kern_el.gplot')
save('E', 'gplot', [], 'E.gplot')

printstats()
savegraph("graph.png")
